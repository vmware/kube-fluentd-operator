require "fluent/plugin"
require "fluent/input"

module Fluent
  module Plugin
    # This is not a plugin. It is used by the extreme validator to just exit with 0
    # when all other plugins have successfully configured themselves
    class JustExit < Input
      Fluent::Plugin.register_input("just_exit", self)

      def configure(conf)
        super
      end

      def start()
        super
        # because input plugins are started last if we get here
        # this means all input/filters are ok. User configs cannot
        # define sources so it's all safe
        Kernel.exit!(0)
      end

      def shutdown()
        super
      end
    end
  end
end

