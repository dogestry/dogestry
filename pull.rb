#!/usr/bin/env ruby

require File.expand_path("../lib/dogestry", __FILE__)

require 'tapp'
require 'pathname'
require 'fileutils'
require 'yajl'

include Dogestry::Sh

@image = ARGV.shift

@work = Pathname("/tmp/dogestry/pull_work").expand_path.tapp(:work)


def parse_name
  @image,@tag = *@image.split(':')

  unless @tag
    @tag = 'latest'
  end
end


def prepare_work
  FileUtils.rm_rf work
  work.mkpath
end


def prepare_image(work)
  finished = false
  walk_refs_from_id(ref_id) {|id, meta|
    # only pull until we hit existing parent in docker
    if local_docker_has_id?(id)
      puts "docker already has #{id}, breaking"
      break
    else
      pull_image(id, work)
    end
  }

  # write repos
  repos = {@image => {@tag => ref_id}}

  (work+'repositories').open('w') do |io|
    Yajl::Encoder.encode(repos, io)
  end
end


def load_image(work)
  Dir.chdir(work) do
    sh!("tar cvf - . | sudo docker load")
  end
end


def walk_refs_from_id(id, &block)
  if meta = image_metadata(id)
    yield(meta['id'], meta)

    if meta['parent']
      walk_refs_from_id(meta['parent'], &block)
    end
  else
    yield nil, {}
  end
end


def local_docker_has_id?(id)
  sh?("sudo docker inspect #{id} > /dev/null")
end




# impl specific
def impl_setup
  @repo = Pathname("/tmp/dogestry/repo").expand_path.tapp(:repo)
  @ref_file = @repo + 'repositories' + @image + @tag
end


def validate_refs
  @ref_file.exist? || raise("image #{@image}:#{@tag} doesn't exist")
end


def ref_id
  @ref_file.read
end


def image_metadata(id)
  Yajl::Parser.parse((image_path(id) + 'json').read)
end


def image_path(id)
  @repo + 'images' + id
end


def pull_image(id, work)
  sh! "rsync -avz #{@repo+'images'+id}/ #{work+id}/"
end


parse_name
impl_setup
validate_refs

prepare_work(@work)
prepare_image(@work)
load_image(@work)
