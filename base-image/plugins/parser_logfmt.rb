# Copyright © 2018 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause

require 'fluent/parser'
require 'logfmt'

module Fluent
  class LogfmtParser < Fluent::Parser
    Fluent::Plugin.register_parser("logfmt", self)

    config_param :strict, :bool, default: false
    def configure(conf)
      super
    end

    def parse(text)
        record = Logfmt.parse(text)
        if @strict
          record.each do |key, val|
            if val == true
              yield Engine.now(), {'message' => text}
              return
            end
          end
        end

        convert_field_type!(record) if @type_converters
        time = record.delete(@time_key)
        if time.nil?
          time = Engine.now
        elsif time.respond_to?(:to_i)
          time = time.to_i
        else
          raise RuntimeError, "The #{@time_key}=#{time} is a bad time field"
        end

        yield time, record
      end
  end
end