#!/usr/bin/env ruby

require File.expand_path("../lib/dogestry", __FILE__)

require 'tapp'
require 'pathname'
require 'fileutils'

@image = ARGV.shift

include Dogestry::Sh

@repo = Pathname("/tmp/dogestry/repo").expand_path.tapp
@work = Pathname("/tmp/dogestry/work").expand_path.tapp

# this could be: local, s3, http etc
def sync
  local_sync
end


def local_sync
  @repo.mkpath
  sh!("rsync -av #{@work}/ #{@repo}/")
end


sh!("./prepare.rb #{@work} #{@image}")
sync
