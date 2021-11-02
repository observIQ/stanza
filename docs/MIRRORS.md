# Creating & Using Mirrors for Stanza Releases

## Creating a Mirror
Mirrors for Stanza can come in two forms:  
1. Hosted websites
2. Local filesystem mirrors

The only requirements for either are creating a directory layout that  
mirrors that of the GitHub releases, such as in the visualization below.  
It is suggested to use an automated synchronization process to manage  
keeping ths up to date, including rewriting the symlink for the latest  
to the highest version number.

### Mirror Tree Visualization
➜  stanza_mirror tree
.
├── download
│   └── v1.2.12
│       ├── stanza-plugins.tar.gz
│       ├── stanza-plugins.zip
│       ├── stanza_darwin_amd64
│       ├── stanza_linux_amd64
│       ├── stanza_linux_arm64
│       ├── stanza_windows_amd64
│       ├── unix-install.sh
│       ├── version.json
│       └── windows-install.ps1
└── latest
    └── download
        ├── stanza-plugins.tar.gz
        ├── stanza-plugins.zip
        ├── stanza_darwin_amd64
        ├── stanza_linux_amd64
        ├── stanza_linux_arm64
        ├── stanza_windows_amd64
        ├── unix-install.sh
        ├── version.json
        └── windows-install.ps1

## Usage Syntax with the Install Script

### Web URL
```shell
# Latest
./unix-install -l http://dl.example.com/some/path
# Specific Version 1.2.12
./unix-install -l http://dl.example.com/some/path -v 1.2.12
```

### File URL
```shell
./unix-install -l file:///Users/username/Downloads/stanza_local
# Specific Version 1.2.12
./unix-install -l file:///Users/username/Downloads/stanza_local -v 1.2.12
```

## Further Information
For further usage information, and other supported flags, please see the [Quick Start Guide](README.md)
