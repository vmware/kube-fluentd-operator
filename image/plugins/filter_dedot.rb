require 'fluent/plugin/filter'

module Fluent::Plugin
  class DedotFilter < Filter

    Fluent::Plugin.register_filter('dedot', self)

    config_param :de_dot, :bool, default: true
    config_param :de_dot_separator, :string, default: '_'

    def initialize
      super
    end

    def configure(conf)
      super

      if @de_dot && @de_dot_separator.include?(".")
        raise Fluent::ConfigError, "Invalid de_dot_separator: cannot be or contain '.'"
      end

      if @de_dot && @de_dot_nested
        log.info "DeDot will recurse nested hashes and arrays"
      end

    end

    def filter(tag, time, record)
      begin
        de_dot(record) if @de_dot
      rescue => e
        router.emit_error_event(tag, time, record, e)
      end
    end

    def de_dot(record)
      newrecord = {}

      record.each do |key, value|
        newkey = key.gsub(/\./, @de_dot_separator)

        # Recurse hashes and arrays:
        if value.is_a? Hash
          value = de_dot value
        elsif value.is_a? Array
          value = value.map { |v| v.is_a?(Hash) ? de_dot(v) : v }
        end

        newrecord[newkey] = value
      end

      newrecord
    end

  end
end