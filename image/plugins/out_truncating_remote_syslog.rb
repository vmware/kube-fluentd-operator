require "fluent/plugin"
require "fluent/plugin/out_remote_syslog"

module Fluent
  module Plugin
    class TagTruncatingRemoteSyslogOutput < Output
      Fluent::Plugin.register_output("truncating_remote_syslog", self)

      def format(tag, time, record)
        begin
          t = truncate_tag(tag)
        rescue
          t = tag[0..31]
        end

        @delegate.format(t, time, record)
      end

      def initialize
        super
        @delegate = Fluent::Plugin::RemoteSyslogOutput.new()
      end

      def configure(conf)
        super
        @delegate.configure(conf)
      end

      def shutdown()
        super
        @delegate.shutdown()
      end

      def write(chunk)
        @delegate.write(chunk)
      end

      def truncate_tag(tag)
        parts = tag.split(/\./)

        if parts[0] == '_proc'
          # this is coming from the exception detector
          parts.shift()
        end

        if parts[0] != 'kube' || parts.length < 4
          # not my business, be nice anyway
          if tag.length <= 32
            return tag
          end
          return tag[0..30] + '*'
        end

        # parts[0] is always 'kube'
        ns = parts[1]
        pod = parts[2]
        container = parts[3]

        total = ns.length + pod.length + container.length + 2
        if total <= 32
          return ns + '.' + pod + '.' + container
        end

        overhead = total - 32
        if pod.length <= overhead
          # unlikely: pod name is shorter than the container name
          t = ns + '.' + pod + '.' + container
          return t[0..30] + '*'
        end
        
        
        t =  ns + '.' + pod[0..-overhead-2] + '*.' + container 
        if t.length > 32
          # that's an extremely long namespace name!
          return t[0..31]
        end

        t
      end
    end
  end
end