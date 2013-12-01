module Dogestry
  module Sh
    extend self

    def sh!(cmd)
      system(cmd)
      $?.tapp.success? || raise("failed to run '#{cmd}'")
    end


    def sh?(cmd)
      system(cmd)
      $?.success?
    end
  end
end
