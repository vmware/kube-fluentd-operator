require 'fluent/plugin/filter'

module Fluent::Plugin
  class EnrichFilter < Filter
    # Register this filter as "passthru"
    Fluent::Plugin.register_filter('extract', self)

#   <rule>
#     key message
#     pattern ^hello-(world)$
#     set type
#     value greet.$1
#   </rule>

    config_section :rule, multi: true, required: false do
        config_param :key, :string
        config_param :pattern do |value|
          if value.start_with?("/") && value.end_with?("/")
            s = value[1..-2]
          else
            $log.warn "You should use \"pattern /#{value}/\" instead of \"pattern #{value}\""
            s = value
          end

          if !s.start_with?("(?")
            # implied dot-matchall unless mode is set
            s = "(?m)" + s
          end
          Regexp.compile(s)
        end
        config_param :set, :string
        config_param :to, :string
       # config_param :type, :string
    end

    def configure(conf)
      super
      # do the usual configuration here
    end

    # def start
    #   super
    #   # Override this method if anything needed as startup.
    # end

    # def shutdown
    #   # Override this method to use it to free up resources, etc.
    #   super
    # end

    def filter(tag, time, record)
      unless @rule.empty?
        @rule.each { |rule|
          key = rule.key
          if record.include?(key) && rule.pattern.match(record[key])
            record[rule.set] = record[key].gsub(rule.pattern, rule.to)
          end
        }
      end

      return record
    end
  end
end