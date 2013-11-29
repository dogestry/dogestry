require 'rubygems'
require 'bundler/setup'

require 'tapp'
require 'pathname'
require 'fileutils'

HERE = File.expand_path("..", __FILE__)

@image = ARGV.shift

def prepare_work
  @work = Pathname("/tmp/dogestry/work").expand_path.tapp

  #FileUtils.rm_rf work

  @work.mkpath

  Dir.chdir(@work)

  Dir.pwd.tapp

  # XXX rm at exit
  #at_exit {
  #}
end


# this could be: local, s3, http etc
def sync
end



#prepare_work
#system("prepare.rb #{@work} #{@image}"
sync
