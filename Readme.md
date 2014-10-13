<a href="https://imgflip.com/i/6v6ii"><img src="https://i.imgflip.com/6v6ii.jpg" title="made at imgflip.com"/></a>

# dogestry

Proof of concept for simple image storage for docker.
This is a simple client you can run where you run docker - and you don't need a registry - it talks directly to s3 (for example). docker save/load is the mechanism used. 

## prerequisites

* [lz4][lz4] -  compiled and on the path
* go 1.2
* docker

Currently, the user running dogestry needs permissions to access the docker socket. [See here for more info][docker-sudo]

The docker connection is local socket (`unix:///var/run/docker.sock`) as default. But is overridable configuring `connection` in `[docker]` entry in `dogestry.cfg`.

## usage

### push

Push the `redis` image and its current tag to the `central` remote. The `central` remote is an alias to a remote defined in `dogestry.cfg`
```
dogestry push central redis
```

Push the `hipache` image to the s3 bucket `ops-goodies` with the key prefix `docker-repo` located in `us-west-2`:
```
dogestry push s3://ops-goodies/docker-repo/?region=us-west-2 hipache
```

### pull

Pull the `hipache` image and tag from the `central`.
```
dogestry pull central hipache
```

And the s3 version: 

```
dogestry pull s3://ops-goodies/docker-repo/?region=us-west-2 hipache
```

### config

Configure dogestry with `dogestry.cfg`. By default it's looked for in `./dogestry.cfg`.

Dogestry can often run without a configuration file, but it's there if you need it.

For example, using the config file, you can set up remote aliases for convenience or specifiy s3 credentials.

However, if you're bootstrapping a system, you might rely on IAM instance profiles for credentials and specify the
remote using its full url. 

### S3

When working with s3, you can use environment variables for credentials, or use signed URLs. The advantage of signed URLs is that you can tightly control the resouce access. 

A common use case if you have a build server building and publishing images via dogestry (needs read write) - but when you deploy - dogestry only needs read access to s3, and can use signed urls (so you don't need any configuration - the URL contains all that is needed to pull the repository): 

```
  Typical S3 Usage:
     export AWS_ACCESS_KEY=ABC
     export export AWS_SECRET_KEY=DEF
     dogestry push s3://<bucket name>/<path name>/?region=us-east-1 <repo name>
     dogestry pull s3://<bucket name>/<path name>/?region=us-east-1 <repo name>
```



## operation

Dogestry push works by
* transforming the output of `docker save` into a local repository in a portable repository format.
  * writing self contained image data.
  * unrolling the repositories json into a directory of files.
* efficiently synchronising the local repository with a remote (e.g. s3, local disk)
  * we only write images not already existing on the remote.

Dogestry pull works by
* resolving the requested image name or id by querying the remote:
  * it could be a mapping from a docker "repostitory:tag" pair to an id.
  * it could be the prefix of an image id which exists on the remote.
* efficiently synchronise images from the remote.
  * walk the ancetry of the requested image until we reach an image that the local docker already has.
  * download the required images.
* preparing a tarball in the format needed by `docker load` and sending it to docker.

## discussion

In my organisation docker will be the way for us to move away from Capistrano's [`cap deploy`][cap].

Capistrano has a number of problems which docker solves neatly or sidesteps entirely. However to make the investment of
time and energy worthwhile in moving away, docker must solve all of the problems Capistrano presents.

It currently does not do this. Luckily most of these blockers are concentrated in the registry approach.

In capistrano (as we use it):
* dependencies are not resolved until during deployment. 
  * If the services hosting these dependencies are down, we're unable to deploy.
  * If these services go down half way through a deploy onto multiple boxes: chaos.
  * This is particularly the case on fresh boxes, `bundle install` is a very expensive and coupled to external service uptime.
* Capistrano as software is complex
  * Maintinaing recipes is difficult
  * Debugging recipes is difficult
  * Testing ditto

Docker's registry doesn't solve these problems.

* The official registry (http://index.docker.io) is centralised.
  * Particularly on a fresh machine, if we can't pull the ubuntu image, we're out of luck.
  * So we need to ensure that images are available from somewhere internal
* Docker's interaction with docker-registry is complex and tightly coupled
  * Tied in with the first problem, there are no guarantees that docker won't go out to the official registry for images. This isn't acceptible for production.
  * It might delegate to the index for auth. Again, in production I would want to actively disable this.
* Setting up docker-registry seems complex
  * It doesn't support secure setups out of the box. There's a suggestion to use basic auth, but no documentation on how to set it up.
  * I've spent a long time trying to work out how to get basic auth working, but haven't cracked it yet!
  * By comparison: docker's single go binary

### enter dogestry

Dogestry aims to solve these particular problems by simplifying the image storage story, while maintaining some of the convenience of
the docker registry.

Dogestry's design aims to support a wide range of dumb and smart transports.

It centres around a common portable repository format.

### synchronisation

Using the new feature for de/serialising self-consistent image histories (`GET /images/<name>/get` and `POST /images/load`) 

* dogestry push - push images from local docker instance to the remote in the portable repo format
* dogestry pull - pull images from the remote into the local docker instance

### remotes

"Remotes" are the external storage locations for the docker images. Dogestry implements transports for each kind of remote, much
like git.

Authentication/authorisation is orthogonal to the concerns of dogestry. It relies on each transport's natural authorisation, such as s3's AWS credentials, or 
unix filesystem permissions. Future transports could use ssh, sftp or the docker registry API.

#### local remote

Dumb transport, synchronises with a directory on the same machine using normal filesystem operations and rsync.

#### s3 remote

Dumb transport, synchronises with an s3 bucket using the s3 api.

#### registry remote (not implemented)

Smart transport, synchronises with an instance of docker-registry.

#### others

Dedicated dogestry server, ssh, other cloud file providers.

### portable repository format

* able to serve as a repository over dumb transports (rsync, s3)

example layout:

images:
```
images/5d4e24b3d968cc6413a81f6f49566a0db80be401d647ade6d977a9dd9864569f/layer.tar
images/5d4e24b3d968cc6413a81f6f49566a0db80be401d647ade6d977a9dd9864569f/VERSION
images/5d4e24b3d968cc6413a81f6f49566a0db80be401d647ade6d977a9dd9864569f/json 
```

To better support eventually-consistent remotes using dumb transports (i.e. s3) The repositories json is unrolled into files (like `.git/refs`)
```
repositories/myapp/20131210     (content: 5d4e24b3d968cc6413a81f6f49566a0db80be401d647ade6d977a9dd9864569f)
repositories/myapp/latest       (content: 5d4e24b3d968cc6413a81f6f49566a0db80be401d647ade6d977a9dd9864569f)
```

#### optional - compression

(**This is switched off for the moment.**)

I've chosen to use lz4 as the compression format as it's very fast and for `layer.tar` still seems to provide reasonable compression ratios. 
There's a [go implementation][golz4] but there's no streaming (i.e. `io.Reader`/`io.Writer`) version and I wouldn't know where to start in converting it.

Given that remotes are generally, well, remote, I don't think it's a stretch to include compression for the portable repository format.

It probably should be optional though.

Currently it's part of the Push/Pull command, but I intend to push the implementation down into the s3 remote.


Lz4 was really impressive compressing layer.tar. Here are some rough numbers performed on a virtualbox vm:

method       | size | compress                                   | decompress
---          | ---  | ---                                        | ---
uncompressed | 848M |                                            | 
gzip         | 288M | 28.2s (real 0m28.279s user 0m27.440s sys 0m0.640s) | 5.9s (real 0m5.862s user 0m4.584s sys 0m1.044s)
lz4          | 397M | 2.7s (real 0m2.697s user 0m0.000s sys 0m0.000s   | 1.4s (real 0m1.473s user 0m0.548s sys 0m0.668s)



#### optional - checksumming

Some remotes support cheap checksumming by default, others don't.

I've implemented checksumming as part of the s3 remote since it turns out that what seems like cheap checksumming (ETag for each s3 object) isn't always the md5 of the object.

## docker changes
* dogestry can work with docker as-is
* at the very least, I'd like to add a flag to enact the zero external dependency requirement (this could be done with e.g. hacking /etc/hosts, but a docker flag would be neater & more pro).
* nice to have would be a refinement of `GET /images/<name>/get` to exclude images already on the remote.
* nice to have would be in-stream de/compression of layer.tar in `GET /images/<name>/get` and `POST /images/load`.
* best of all would be to integrate some different registry approaches into docker.

## TODO

- more tests.
- more remotes.
- more tag operations
- tree pruning


## conclusion

Although I'd like docker's external image storage approach to be more flexible and less complex, I can support what I need reasonably efficiently with current docker features.

Dogestry's main aim is to support my use-case, but I hope that this code stimulates some discussion on the subject.


[cap]: https://github.com/capistrano/capistrano
[golz4]: https://github.com/bkaradzic/go-lz4
[lz4]: https://code.google.com/p/lz4/
[docker-sudo]: https://docs.docker.io/en/latest/use/basics/#sudo-and-the-docker-group


## License

The MIT License (MIT)

Copyright (c) 2014 Blake eLearning

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
