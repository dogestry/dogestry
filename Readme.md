# dogestry

Proof of concept for simple image storage for docker.

## thesis

In my organisation docker will be the way for us to move away from [`cap deploy`][cap]

* When deploying with `cap`, a window of risk in opened.
* resource wastage and contention
  * The next release of code is prepared to run on the same machine as the running code.
  * If we have N servers, this is performed in parallel on N servers.
  * This means that the running production app can be affected by deployments which use too much CPU or RAM.
* external dependencies
  * Our apps are ruby on rails.
  * Part of code preparation is `bundle install` which satisfied code dependencies.
  * `bundle install` goes out to github.
  * github is partially down.
  * 7/12 servers get deployed and restarted. 5/12 do not.

The promise of docker is to take all this preparation offline. This solves resource wastage and contention but as it stands
we're still exposed to problems with external dependencies.

Specifically, using docker relies on the central registry being up.

* `docker run myregistry.mycompany.com/myapp:20131210`
* `myregistry.mycompany.com/myapp` is pulled successfully, but relies on `ubuntu` somewhere in its history.
* `index.docker.io` is unreachable for some reason.
* bummer.

### other problems

* Simple but secure docker-registry setup is complex; I can't it working with basic auth.

## solution

### synchronisation

Using the new feature for de/serialising self-consistent image histories (`GET /images/<name>/get` and `POST /images/load`) 

* dogestry push - push images from local docker instance to the remote in the portable repo format
* dogestry pull - pull images from the remote into the local docker instance

### remotes

"Remotes" are the external storage locations for the docker images.

#### local remote

Dumb transport, synchronises with a directory on the same machine using normal filesystem operations and rsync.

#### s3 remote

Dumb transport, synchronises with s3 using the s3 api.

#### registry remote (not implemented)

Smart transport, synchronises with the docker-registry api.

### portable repository format

* able to serve as a repository over dumb transports (rsync, s3)
* adds compression to `layer.tar`

example layout

images:
```
images/5d4e24b3d968cc6413a81f6f49566a0db80be401d647ade6d977a9dd9864569f/layer.tar.lz4
images/5d4e24b3d968cc6413a81f6f49566a0db80be401d647ade6d977a9dd9864569f/VERSION
images/5d4e24b3d968cc6413a81f6f49566a0db80be401d647ade6d977a9dd9864569f/json 
```

repositories:
```
repositories/myapp/20131210     (content: 5d4e24b3d968cc6413a81f6f49566a0db80be401d647ade6d977a9dd9864569f)
repositories/myapp/latest       (content: 5d4e24b3d968cc6413a81f6f49566a0db80be401d647ade6d977a9dd9864569f)
```

## docker changes
* dogestry can work with docker as-is
* at the very least, I'd like to add a flag to enact the zero external dependency requirement (this could be done with e.g. hacking /etc/hosts, but a docker flag would be neater & more pro).
* nice to have would be a refinement of `GET /images/<name>/get` to exclude images already on the remote.
* nice to have would be in-stream de/compression of layer.tar in `GET /images/<name>/get` and `POST /images/load`.
* best of all would be to integrate some different registry approaches.


[cap]: https://github.com/capistrano/capistrano
