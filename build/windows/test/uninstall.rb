agent_home="C:/Program Files/stanza"

describe service('stanza') do
    it { should_not be_enabled }
    it { should_not be_installed }
    it { should_not be_running }
end
