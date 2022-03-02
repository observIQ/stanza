[
    '/opt/observiq/stanza',
    '/opt/observiq/stanza/plugins',
].each do |dir|
    describe file(dir) do
        its('mode') { should cmp '0755' }
        its('owner') { should eq 'stanza' }
        its('group') { should eq 'stanza' }
        its('type') { should cmp 'directory' }
    end
end

describe file('/opt/observiq/stanza/stanza.db') do
    its('mode') { should cmp '0600' }
    its('owner') { should eq 'stanza' }
    its('group') { should eq 'stanza' }
    its('type') { should cmp 'file' }
end

describe file('/opt/observiq/stanza/stanza.log') do
    its('mode') { should cmp '0644' }
    its('owner') { should eq 'stanza' }
    its('group') { should eq 'stanza' }
    its('type') { should cmp 'file' }
end

describe file('/opt/observiq/stanza/config.yaml') do
    its('mode') { should cmp '0640' }
    its('owner') { should eq 'stanza' }
    its('group') { should eq 'stanza' }
    its('type') { should cmp 'file' }
end

describe file('/usr/bin/stanza') do
    its('mode') { should cmp '0755' }
    its('owner') { should eq 'root' }
    its('group') { should eq 'root' }
    its('type') { should cmp 'file' }
end

# Stanza can install on Centos 6 but we do not support the
# centos 6 init system.
if !os[:release].start_with?('6')
    describe systemd_service('stanza') do
        it { should be_installed }
        it { should be_enabled }
        it { should be_running }
    end
end
