# Creating & Using Mirrors for Stanza Releases

## Creating a Mirror
Mirrors for Stanza can come in two forms:  
1. Hosted websites
2. Local filesystem mirrors

The only requirements for either are creating a directory layout that  
mirrors that of the GitHub releases, ie:  
Example of web URL for latest version: `http://dl.example.com/some/path/latest/download/`  
Example of web URL for version 1.2.13: `http://dl.example.com/some/path/download/v1.2.13/`  
Example of web URL for version 1.2.12: `http://dl.example.com/some/path/download/v1.2.12/`  
Example of file URL for latest version: `file:///Users/username/Downloads/stanza_local/latest/download/`  
Example of file URL for version 1.2.13: `file:///Users/username/Downloads/stanza_local/download/v1.2.13`
Example of file URL for version 1.2.12: `file:///Users/username/Downloads/stanza_local/download/v1.2.12`


## Usage Syntax with the Install Script

### Web URL
```shell
./unix-install -l http://dl.example.com/some/path/latest/download
```

### File URL
```shell
./unix-install -l file:///Users/username/Downloads/stanza_local/latest/download
```

## Further Information
For further usage information, and other supported flags, please see the base [README.md](README.md)
