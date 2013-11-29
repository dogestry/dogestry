require 'rubygems'
require 'bundler/setup'

require 'tapp'
require 'pathname'
require 'fileutils'


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



prepare_work
system("prepare.rb #{@work} #{@image}"
sync
