new-module -name LogAgentInstall -scriptblock {

  # Constants
  $DEFAULT_WINDOW_TITLE = $host.ui.rawui.WindowTitle
  $DEFAULT_INSTALL_PATH = 'C:\'
  $DOWNLOAD_BASE = "https://github.com/observiq/carbon/releases/latest/download"
  $SERVICE_NAME = 'carbon_log_agent'
  $INDENT_WIDTH = '  '
  $MIN_DOT_NET_VERSION = '4.5'

  # Functions
  function Set-Variables {
    if ($host.name -NotMatch "ISE") {
      $IS_PS_ISE = $false
      $script:DEFAULT_FG_COLOR = $host.ui.rawui.ForegroundColor
      $script:DEFAULT_BG_COLOR = $host.ui.rawui.BackgroundColor
    }
    else {
      $script:IS_PS_ISE = $true
      $script:DEFAULT_FG_COLOR = [System.ConsoleColor]::White
      $script:DEFAULT_BG_COLOR = $psIse.Options.ConsolePaneBackgroundColor
    }
  }

  function Add-Indent {
    $script:indent = "${script:indent}$INDENT_WIDTH"
  }

  function Remove-Indent {
    $script:indent = $script:indent -replace "^$INDENT_WIDTH", ''
  }

  # Takes a variable amount of alternating strings and their colors.
  # An empty-string color uses the default text color.
  # The last given string does not require a color (uses default)
  # e.g.: string1 color1 string2 color2 string3
  function Show-ColorText {
    Write-Host "$indent" -NoNewline
    for ($i = 0; $i -lt $args.count; $i++) {
      $message = $args[$i]
      $i++
      $color = $args[$i]
      if (!$color) {
        $color = $script:DEFAULT_FG_COLOR
      }
      Write-Host "$message" -ForegroundColor "$color" -NoNewline
    }
    Write-Host ""
  }

  function Show-Separator {
    Show-ColorText "============================================" $args
  }

  function Show-Header {
    $message, $color = $args
    Show-ColorText ""
    Show-Separator
    Show-ColorText '| ' '' "$message" $color
    Show-Separator
  }

  function Set-Window-Title {
    $host.ui.rawui.windowtitle = "Log Agent Install"
  }

  function Restore-Window-Title {
    $host.ui.rawui.windowtitle = $DEFAULT_WINDOW_TITLE
  }

  function Complete {
    Show-ColorText "Complete" DarkGray
  }

  function Succeeded {
    Add-Indent
    Show-ColorText "Succeeded!" Green
    Remove-Indent
  }

  function Failed {
    Add-Indent
    Show-ColorText "Failed!" Red
    Remove-Indent
  }

  function Show-Usage {
    Add-Indent
    Show-ColorText 'Options:'
    Add-Indent

    Show-ColorText ''
    Show-ColorText '-y, --accept_defaults'
    Add-Indent
    Show-ColorText 'Accepts all default values for installation.' DarkCyan
    Remove-Indent

    Show-ColorText ''
    Show-ColorText '-d, --download_url'
    Add-Indent
    Show-ColorText 'Defines the download url of the agent.' DarkCyan
    Show-ColorText 'If not provided, this will default to the standard location.' DarkCyan
    Remove-Indent

    Show-ColorText ''
    Show-ColorText '-i, --install_dir'
    Add-Indent
    Show-ColorText 'Defines the install directory of the agent.' DarkCyan
    Show-ColorText 'If not provided, this will default to C:/observiq.' DarkCyan
    Remove-Indent

    Show-ColorText ''
    Show-ColorText '-u, --service_user'
    Add-Indent
    Show-ColorText 'Defines the service user that will run the agent.' DarkCyan
    Show-ColorText 'If not provided, this will default to the current user.' DarkCyan
    Remove-Indent

    Remove-Indent
    Remove-Indent
  }

  function Exit-Error {
    if ($indent) { Add-Indent }
    $line_number = $args[0]
    $message = ""
    if ($args[1]) { $message += "`n${indent}- [Issue]: $($args[1])" }
    if ($args[2]) { $message += "`n${indent}- [Resolution]: $($args[2])" }
    if ($args[3]) { $message += "`n${indent}- [Help Link]: $($args[3])" }
    if ($args[4]) {
      $message += "`n${indent}- [Rerun]: $($args[4])"
    }
    elseif ($script:rerun_command) {
      $message += "`n${indent}- [Rerun]: $script:rerun_command"
    }
    throw "Error (windows_install.ps1:${line_number}): $message"
    if ($indent) { Remove-Indent }
  }

  function Get-AbsolutePath ($path) {
    $path = [System.IO.Path]::Combine(((Get-Location).Path), ($path))
    $path = [System.IO.Path]::GetFullPath($path)
    return $path;
  }

  function Request-Confirmation ($default = "y") {
    if ($default -eq "n") {
      Write-Host -NoNewline "y/"
      Write-Host -NoNewline -ForegroundColor Red "[n]: "
    }
    else {
      Write-Host -NoNewline -ForegroundColor Green "[y]"
      Write-Host -NoNewline "/n: "
    }
  }

  # This will check for all required conditions
  # before executing an installation.
  function Test-Prerequisites {
    Show-Header "Checking Prerequisites"
    Add-Indent
    Test-PowerShell
    Test-Architecture
    Test-DotNet
    Complete
    Remove-Indent
  }

  # This will ensure that the script is executed
  # in the correct version of PowerShell.
  function Test-PowerShell {
    Show-ColorText  "Checking PowerShell... "
    if (!$ENV:OS) {
      Failed
      Exit-Error $MyInvocation.ScriptLineNumber 'Install script was not executed in PowerShell.'
    }

    $script:PSVersion = $PSVersionTable.PSVersion.Major
    if ($script:PSVersion -lt 3) {
      Failed
      Exit-Error $MyInvocation.ScriptLineNumber 'Your PowerShell version is not supported.' 'Please update to PowerShell 3+.'
    }
    Succeeded
  }


  # This will ensure that the CPU Architecture is supported.
  function Test-Architecture {
    Show-ColorText 'Checking CPU Architecture... '
    if ([System.IntPtr]::Size -eq 4) {
      Failed
      Exit-Error $MyInvocation.ScriptLineNumber '32-bit Operating Systems are currently not supported.'
    }
    else {
      Succeeded
    }
  }

  # This will ensure that the version of .NET is supported.
  function Test-DotNet {
    Show-ColorText 'Checking .NET Framework Version...'
    if ([System.Version](Get-ChildItem 'HKLM:\SOFTWARE\Microsoft\NET Framework Setup\NDP' -recurse | Get-ItemProperty -name Version, Release -EA 0 | Where-Object { $_.PSChildName -match '^(?!S)\p{L}' } | Sort-Object -Property Version -Descending | Select-Object Version -First 1).version -ge [System.Version]"$MIN_DOT_NET_VERSION") {
      Succeeded
    }
    else {
      Failed
      Exit-Error $MyInvocation.ScriptLineNumber ".NET Framework $MIN_DOT_NET_VERSION is required." "Install .NET Framework $MIN_DOT_NET_VERSION or later"
    }
  }

  # This will set the values of all installation variables.
  function Set-InstallVariables {
    Show-Header "Configuring Installation Variables"
    Add-Indent
    Set-Defaults
    Set-DownloadURL
    Set-InstallDir
    Set-HomeDir
    Set-BinaryLocation
    Set-ServiceUser

    Complete
    Remove-Indent
  }

  # This will prompt a user to use default values. If yes, this
  # will set the install_dir, agent_name, and service_user to their
  # default values.
  function Set-Defaults {
    If ( !$script:accept_defaults ) {
      Write-Host -NoNewline "${indent}Accept installation default values? "
      Request-Confirmation
      $script:accept_defaults = Read-Host
    }
    Else {
      $accepted_defaults_via_args = "true"
    }

    Switch ( $script:accept_defaults.ToLower()) {
      { ($_ -in "n", "no") } {
        return
      }
      default {
        If (!$accepted_defaults_via_args) { Write-Host -NoNewline "${indent}" }
        Show-ColorText "Using default installation values" Green
        If (!$script:install_dir) {
          $script:install_dir = $DEFAULT_INSTALL_PATH
        }
        If (!$script:service_user) {
          $script:service_user = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
        }
      }
    }
  }

  # This will set the install path of the agent. If not provided as a flag,
  # the user will be prompted for this information.
  function Set-InstallDir {
    Show-ColorText 'Setting install directory...'
    Add-Indent
    If ( !$script:install_dir ) {
      Write-Host -NoNewline "${indent}Install path   ["
      Write-Host -NoNewline -ForegroundColor Cyan "$DEFAULT_INSTALL_PATH"
      Write-Host -NoNewline ']: '
      $script:install_dir = Read-Host
      If ( !$script:install_dir ) {
        $script:install_dir = $DEFAULT_INSTALL_PATH
      }
      $script:install_dir = Get-AbsolutePath($script:install_dir)
    }
    else {
      $script:install_dir = [System.IO.Path]::GetFullPath($script:install_dir)
    }

    If (-Not (Test-Path $script:install_dir) ) {
      New-Item -ItemType directory -Path $script:install_dir | Out-Null
    }

    Show-ColorText 'Using install directory: ' '' "$script:install_dir" DarkCyan
    Remove-Indent
  }

  # This will set download url for the agent. If not provided as a flag,
  # this will be constructed from the agent version and download base.
  function Set-DownloadURL {
    Show-ColorText 'Configuring download url...'
    Add-Indent
    if ( !$script:download_url ) {
      $script:download_url = "$DOWNLOAD_BASE/carbon_windows_amd64"
    }
    Show-ColorText "Using download url: " '' "$script:download_url" DarkCyan
    Remove-Indent
  }

  # This will set the home directory of the agent based on
  # the install directory provided.
  function Set-HomeDir {
    Show-ColorText 'Setting home directory...'
    Add-Indent
    $script:agent_home = "{0}observiq" -f $script:install_dir

    If (-Not (Test-Path $script:agent_home) ) {
      New-Item -ItemType directory -Path $agent_home | Out-Null
    }

    Show-ColorText "Using home directory: " '' "$script:agent_home" DarkCyan
    Remove-Indent
  }

  # This will set the path for the agent binary.
  function Set-BinaryLocation {
    Show-ColorText 'Setting binary location...'
    Add-Indent

    $script:binary_location = Get-AbsolutePath("$script:agent_home\carbon.exe")
    Show-ColorText "Using binary location: " '' "$script:binary_location" DarkCyan
    Remove-Indent
  }

  # This will set the user that will run the agent as a service.
  # If not provided as a flag, this will default to the current user.
  function Set-ServiceUser {
    Show-ColorText 'Setting service user...'
    Add-Indent
    If (!$script:service_user ) {
      $current_user = [System.Security.Principal.WindowsIdentity]::GetCurrent().Name
      $script:service_user = $current_user
    }
    Show-ColorText "Using service user: " '' "$script:service_user" DarkCyan
    Remove-Indent
  }

  # This will set user permissions on the install directory.
  function Set-Permissions {
    Show-Header "Permissions"
    Add-Indent
    Show-ColorText "Setting file permissions for NetworkService user..."

    try {
      $Account = New-Object System.Security.Principal.NTAccount "NT AUTHORITY\NetworkService"
      $InheritanceFlag = [System.Security.AccessControl.InheritanceFlags]::ContainerInherit -bor [System.Security.AccessControl.InheritanceFlags]::ObjectInherit
      $PropagationFlag = 0
      $NewAccessRule = New-Object Security.AccessControl.FileSystemAccessRule $Account, "Modify", $InheritanceFlag, $PropagationFlag, "Allow"
      $FolderAcl = Get-Acl $script:agent_home
      $FolderAcl.SetAccessRule($NewAccessRule)
      $FolderAcl | Set-Acl $script:agent_home
      Succeeded
    }
    catch {
      Failed
      Exit-Error $MyInvocation.ScriptLineNumber "Unable to set file permissions for NetworkService user: $($_.Exception.Message)"
    }

    try {
      $Account = New-Object System.Security.Principal.NTAccount "$script:service_user"

      # First, ensure modify permissions on the install path
      Show-ColorText 'Checking for ' '' 'Modify' Yellow ' permissions...'
      Add-Indent
      $ModifyValue = [System.Security.AccessControl.FileSystemRights]::Modify -as [int]
      $FolderAcl = Get-Acl $script:agent_home

      $UserHasModify = $FolderAcl.Access | Where-Object { ($_.FileSystemRights -band $ModifyValue) -eq $ModifyValue -and $_.IdentityReference -eq $Account }
      if (-not $UserHasModify) {
        Show-ColorText 'Modify' Yellow ' permissions not found for ' '' "$Account" DarkCyan
        Remove-Indent
        Show-ColorText 'Granting permissions...'
        $NewAccessRule = New-Object Security.AccessControl.FileSystemAccessRule $Account, "Modify", $InheritanceFlag, $PropagationFlag, "Allow"
        $FolderAcl.SetAccessRule($NewAccessRule)
        $FolderAcl | Set-Acl $script:agent_home
        Succeeded
      }
      else {
        Show-ColorText "$Account" DarkCyan ' already possesses ' "" 'Modify' Yellow ' permissions on the install directory.'
        Remove-Indent
      }
    }
    catch {
      Show-ColorText "Unable to verify user permissions: $($_.Exception.Message)" Yellow
    }
    Complete
    Remove-Indent
  }

  # This will download the agent binary to the binary location.
  function Get-AgentBinary {
    Show-Header "Downloading AgentBinary"
    Add-Indent
    Show-ColorText 'Downloading binary. Please wait...'
    Show-ColorText "$INDENT_WIDTH$script:download_url" DarkCyan ' -> ' '' "$script:binary_location" DarkCyan
    try {
      $WebClient = New-Object System.Net.WebClient
      $WebClient.DownloadFile($script:download_url, $script:binary_location)
      Complete
    }
    catch {
      Failed
      $error_message = $_.Exception.Message -replace 'Exception calling.*?: ', ''
      Exit-Error $MyInvocation.ScriptLineNumber "Failed to download agent binary: $error_message"
    }
    Remove-Indent
  }

  # This will ensure that any previous installations are removed.
  function Assert-CleanInstall {
    Show-Header 'Ensuring Clean Installation'
    Add-Indent
    Remove-AgentService

    if ( Test-Path "$script:agent_home" ) {
      try {
        Show-ColorText 'Previous installation detected at ' '' "$script:agent_home" DarkCyan '. Removing...'
        Get-ChildItem "$script:agent_home" -Recurse | Remove-Item -Recurse -Force -ErrorAction Stop
        Show-ColorText 'Previous installation files removed.'
      }
      catch {
        Exit-Error $MyInvocation.ScriptLineNumber "Could not remove $script:agent_home: $($_.Exception.Message)" 'Please ensure you have permission to remove this directory and its files.'
      }
    }
    else {
      Show-ColorText 'Clean installation path detected. (' '' "$script:agent_home" DarkCyan ')'
    }
    Complete
    Remove-Indent
  }

  # This will remove the agent service.
  function Remove-AgentService {
    $service = Get-Service $SERVICE_NAME -ErrorAction SilentlyContinue
    If ($service) {
      Show-ColorText 'Previous ' '' "$SERVICE_NAME" DarkCyan ' service detected.'
      If ($service.Status -eq 'Running') {
        Show-ColorText 'Stopping ' '' "$SERVICE_NAME" DarkCyan '...'
        if ($script:PSVersion -ge 5) {
          Stop-Service $SERVICE_NAME -NoWait -Force | Out-Null
        }
        else {
          Stop-Service $SERVICE_NAME | Out-Null
        }
        Show-ColorText "Service Stopped."
      }
      Show-ColorText 'Removing ' '' "$SERVICE_NAME" DarkCyan ' service...'
      (Get-WmiObject win32_service -Filter "name='$SERVICE_NAME'").delete() | Out-Null

      If ( $LASTEXITCODE -eq 0 ) {
        # Sleep 5 seconds to give time for the service to be fully deleted
        sleep -s 5
        Show-ColorText 'Previous service removed.'
      }
    }
  }

  # This will create the agent config.
  function New-AgentConfig {
    Show-Header "Configuring Agent"
    Add-Indent
    $config_file = "$script:agent_home\config.yaml"
    Show-ColorText 'Writing config file: ' '' "$config_file" DarkCyan
    try {
      Write-Config $config_file
    }
    catch {
      Exit-Error $MyInvocation.ScriptLineNumber "Failed to write config file: $($_.Exception.Message)" 'Please ensure you have permission to create this file.'
    }

    Complete
    Remove-Indent
  }

  # This will write the agent config.
  function Write-Config {
    @"
# pipeline:
#   - id: example_input
#     type: generate_input
#     count: 1
#     record:
#       message: example log
#     output: example_stdout
#
#   - id: example_stdout
#     type: stdout
"@ > $args
  }

  # This will create a new agent service.
  function New-AgentService {
    Show-Header "Configuring Windows Service"
    Add-Indent
    Install-AgentService
    Start-AgentService
    Complete
    Remove-Indent
  }

  # This will install the agent service.
  function Install-AgentService {
    Show-ColorText 'Installing ' '' "$SERVICE_NAME" DarkCyan ' service...'

    $service_params = @{
      Name           = "$SERVICE_NAME"
      DisplayName    = "$SERVICE_NAME"
      BinaryPathName = "$script:binary_location --config $script:agent_home\config.yaml --log_file $script:agent_home\$SERVICE_NAME.log --database $script:agent_home\$SERVICE_NAME.db --plugin_dir $script:agent_home\plugins"
      Description    = "Monitors and processes logs."
      StartupType    = "Automatic"
    }

    try {
      New-Service @service_params -ErrorAction Stop | Out-Null
    }
    catch {
      Exit-Error $MyInvocation.ScriptLineNumber "Failed to install the $SERVICE_NAME service." 'Please ensure you have permission to install services.'
    }


    $script:startup_cmd = "net start `"$SERVICE_NAME`""
    $script:shutdown_cmd = "net stop `"$SERVICE_NAME`""
    $script:autostart = "Yes"
  }

  # This will start the agent service.
  function Start-AgentService {
    Show-ColorText 'Starting service...'
    try {
      Start-Service -name $SERVICE_NAME -ErrorAction Stop
    }
    catch {
      $script:START_SERVICE = $FALSE
      Show-ColorText "Warning: An error prevented service startup: $($_.Exception.Message)" Yellow
      Show-ColorText "A restart may be required to start the service on some systems." Yellow
    }
  }

  # This will finish the install by printing out the results.
  function Complete-Install {
    Show-AgentInfo
    Show-InstallComplete
  }

  # This will display information about the agent after install.
  function Show-AgentInfo {
    Show-Header 'Agent Information'
    Add-Indent
    Show-ColorText 'Install Path:  ' '' "$script:agent_home" DarkCyan
    Show-ColorText 'Start On Boot: ' '' "$script:autostart" DarkCyan
    Show-ColorText 'Start Command: ' '' "$script:startup_cmd" DarkCyan
    Show-ColorText 'Stop Command:  ' '' "$script:shutdown_cmd" DarkCyan
    Complete
    Remove-Indent
  }

  # This will provide a user friendly message after the installation is complete.
  function Show-InstallComplete {
    Show-Header 'Installation Complete!' Green
    Add-Indent
    if ( $script:START_SERVICE ) {
      Show-ColorText "Your agent is installed and running.`n" Green
    }
    else {
      Show-ColorText 'Your agent is installed but not running.' Green
      Show-ColorText "Please restart to complete service installation.`n" Yellow
    }
    Remove-Indent
  }

  function Main {
    [cmdletbinding()]
    param (
      [Alias('y', 'accept_defaults')]
      [string]$script:accept_defaults,

      [Alias('d', 'download_url')]
      [string]$script:download_url,

      [Alias('i', 'install_dir')]
      [string]$script:install_dir = '',

      [Alias('u', 'service_user')]
      [string]$script:service_user,

      [Alias('h', 'help')]
      [switch]$script:help
    )
    try {
      Set-Variables
      # Variables which should be reset if the user calls Log-Agent-Install without redownloading script
      $script:indent = ''
      $script:START_SERVICE = $TRUE
      if ($MyInvocation.Line -match 'Log-Agent-Install.*') {
        $script:rerun_command = $matches[0]
      }
      if ($PSBoundParameters.ContainsKey("script:help")) {
        Show-Usage
      }
      else {
        Set-Window-Title
        Show-Separator
        Test-Prerequisites
        Set-InstallVariables
        Set-Permissions
        Assert-CleanInstall
        Get-AgentBinary
        New-AgentConfig
        New-AgentService
        Complete-Install
        Show-Separator
      }
      $exited_success = $true
    }
    catch {
      if ($_.Exception.Message) {
        if ($script:IS_PS_ISE) {
          $psIse.Options.ConsolePaneBackgroundColor = $script:DEFAULT_BG_COLOR
        }
        else {
          $host.ui.rawui.BackgroundColor = $script:DEFAULT_BG_COLOR
        }

        Show-ColorText "$_" Red
      }
      $exited_error = $true
    }
    finally {
      Restore-Window-Title
      if (!$exited_success -and !$exited_error) {
        # Write-Host is required here; anything that uses the pipeline will block
        # and not print if the user cancels with ctrl+c.
        Write-Host "`nScript canceled by user.`n" -ForegroundColor Yellow
      }
    }
  }

  set-alias 'Log-Agent-Install' -value Main
  Export-ModuleMember -Function 'Main', 'Get-ProjectMetadata' -alias 'Log-Agent-Install'
} | Out-Null
