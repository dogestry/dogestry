require 'rubygems'
require 'bundler/setup'

require 'tapp'
require 'pathname'
require 'fileutils'
require 'yajl'

@work = Pathname(ARGV.shift).expand_path
@image = ARGV.shift

def save_image
  #system("sudo docker save #{@image} | tar xf - --no-same-permissions".taputs)
end


def handle_images
  images = @work.join('images')

  # put images in place
  images.mkpath

  paths = Pathname.glob(@work+'*').select {|path|
    path.directory? && path.basename.to_s[/^[A-Fa-f0-9]{40}/]
  }.tapp.each {|path|
    path.chmod 0700
    system("chmod 0600 #{path+'*'}")
    FileUtils.mv path, images
  }
end


def handle_repos
  repositories = @work.join('repositories')
  repositories_json = @work.join('repositories.json')

  # split out repo

  if repositories.exist? && !repositories.directory?
    repositories.chmod 0600
    FileUtils.mv repositories, repositories_json
  end

  if repositories_json.exist?
    repos = Yajl::Parser.parse(repositories_json.read).tapp
  else
    repos = {}
  end

  repositories.mkpath
  repositories.chmod 0700

  path_and_hashes = repos.select {|name,value| !name[/[^A-Za-z0-9\-\/]+/]}.flat_map do |name,v|
    v.map {|tag,hash| [ repositories + name + tag, hash ]}
  end
  
  path_and_hashes.each do |path, hash|
    path.dirname.mkpath
    path.open('w') do |f|
      f << hash
    end
  end

  repositories_json.unlink
end


save_image
handle_images
handle_repos
