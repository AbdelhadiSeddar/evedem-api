if($args.Count -eq 0) {
    Write-Error -Message "No Action to commit Use exec.ps1 build|run|release "
    exit
}
if (Test-Path ".\bin") {
    if(Test-Path ".\bin\main.exe") {
        Remove-Item -Path ".\bin\main.exe"
    }
} else {
    New-Item -Path “.\bin” -ItemType Directory
}

$envpath = ""
if(Test-Path ".\.env") {
    $envpath = ".\.env"
} else {
    if ( Test-Path ".\bin\.env" ) {
        $envpath = ".\bin\.env"
    } else {
        Write-Error -Message "No .env file in this directory or in 'bin' directory, See README.md "
        exit
    }
}

get-content $envpath | foreach {
    $name, $value = $_.split('=')
    $key, $extra = $name.split(' ')

    if ($key -ne "#" -and $name -ne "") {
        $val, $extra = $value.split('#')
        $val = $val -replace '"'
        set-content env:$key $val
    }
}

$relpath = ""
if(Test-Path ".\RELEASE") {
    $relpath = ".\RELEASE"
} else {
    Write-Error -Message "No RELEASE file in this directory"
}

get-content $relpath | foreach {
    $name, $value = $_.split('=')
    $key, $extra = $name.split(' ')

    if ($key -ne "#" -and $name -ne "") {
        $val, $extra = $value.split('#')
        $val = $val -replace '"'
        set-content env:$key $val
    }
}

$act = $args[0].ToString().ToLower()

if($act -eq "build" -or $act -eq "run")
{    
    go build -o .\bin\main.exe .\cmd\api\main.go
}

if($act -eq "run")
{
    .\bin\main.exe
}

if($act -eq "release")
{    
    set-content env:GOARCH 386
    go build -o .\bin\evdem-api-x86.exe .\cmd\api\main.go

    ISCC.exe .\installer_x86.iss

    set-content env:GOARCH amd64
    go build -o .\bin\evdem-api-x64.exe .\cmd\api\main.go
    
    ISCC.exe .\installer_x64.iss
}

