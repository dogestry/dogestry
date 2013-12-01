require 'pathname'

require 'rubygems'
require 'bundler/setup'

require 'tapp'

module Dogestry
  def self.root
    Pathname("../..").expand_path(__FILE__)
  end

  autoload :Sh, "dogestry/sh"
end

$LOAD_PATH.unshift Dogestry.root+'lib'
