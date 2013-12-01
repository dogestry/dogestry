#!/usr/bin/env ruby

require File.expand_path("../lib/dogestry", __FILE__)

require 'tapp'
require 'pathname'
require 'fileutils'
require 'yajl'

include Dogestry::Sh

@work = Pathname(ARGV.shift).expand_path
@image = ARGV.shift



def save_image_tar
  FileUtils.rm_rf @work
  @work.mkpath

  Dir.chdir(@work) do
    sh!("sudo docker save #{@image} | tar xf -".taputs)
    sh!("find . -type d -exec chmod 0700 {} \\;")
    sh!("find . -type f -exec chmod 0600 {} \\;")
  end
end


def move_images
  images = @work.join('images')

  # put images in place
  images.mkpath

  paths = Pathname.glob(@work+'*').select {|path|
    path.directory? && path.basename.to_s[/^[A-Fa-f0-9]{40}/]
  }.tapp.each {|path|
    FileUtils.mv path.to_s, (images + path.basename).to_s
  }
end


def unroll_repositories
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


save_image_tar
move_images
unroll_repositories
