# Copyright © 2018 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause

require "fluent/plugin"
require "fluent/input"

module Fluent
  module Plugin
    # This is not a real plugin. It is used by the extreme validator to just exit with 0
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
        # https://github.com/fluent/fluentd/blob/v1.2.2/lib/fluent/root_agent.rb#L171
        Fluent::Supervisor.cleanup_resources
        Kernel.exit!(0)
      end

      def shutdown()
        super
      end
    end
  end
end

