agent_home="C:/Program Files/stanza"

[
    "#{agent_home}/plugins"
].each do |dir|
    describe file(dir) do
        it { should exist }
        it { should be_directory }
    end
end

[
    "#{agent_home}/config.yaml",
    "#{agent_home}/stanza.log",
    "#{agent_home}/stanza.db",
    "#{agent_home}/plugins/aerospike.yaml",
    "#{agent_home}/plugins/microsoft_iis.yaml",
    "#{agent_home}/plugins/zookeeper.yaml"
].each do |file|
    describe file(file) do
        it { should exist }
        it { should be_file }
    end
end

describe service('stanza') do
    it { should be_installed }
    it { should be_enabled }
    it { should be_running }
end
