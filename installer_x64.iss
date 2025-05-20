[Setup]
AppName=Evdem API
AppVersion=1.0.0
DefaultDirName={commonpf64}\Evdem\API
DefaultGroupName=Evdem
OutputBaseFilename=evdem_api_installer_x64
Compression=lzma
SolidCompression=yes

[Files]
Source: "bin\evdem-api-x64.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "example.env"; DestDir: "{app}"; Flags: ignoreversion
Source: "start_server.bat"; DestDir: "{app}"; Flags: ignoreversion
Source: "edit_env.bat"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{group}\Start Server"; Filename: "{app}\start_server.bat"; WorkingDir: "{app}"; IconFilename: "{app}\evdem-api.exe"; IconIndex: 0
Name: "{group}\Edit Env File"; Filename: "{app}\edit_env.bat"; WorkingDir: "{app}"; IconFilename: "{app}\evdem-api.exe"; IconIndex: 0
Name: "{group}\Uninstall Evdem API"; Filename: "{uninstallexe}"

[Run]
Filename: "{app}\edit_env.bat"; Description: "Edit Envirenment Variables File"; Flags: nowait postinstall skipifsilent
Filename: "{app}\start_server.bat"; Description: "Start Evdem API {%VERSION}"; Flags: nowait postinstall skipifsilent

[UninstallDelete]
Type: files; Name: "{app}\evdem-api.exe"

[Code]
procedure CurStepChanged(CurStep: TSetupStep);
begin
  if CurStep = ssPostInstall then
  begin
    RenameFile(ExpandConstant('{app}\evdem-api-x64.exe'), ExpandConstant('{app}\evdem-api.exe'));
    RenameFile(ExpandConstant('{app}\example.env'), ExpandConstant('{app}\.env'));
  end;
end;