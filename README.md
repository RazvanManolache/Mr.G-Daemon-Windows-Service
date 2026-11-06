# Mr.G-Daemon-Windows-Service
## GitHub builds

Open **Actions → Windows service build → Run workflow**, then download
`MrG-Daemon-win-x64` from the completed run. Pushes and pull requests also build.
The ZIP includes the EXE, configuration examples, source commit and SHA-256 checksum.

Extract to a permanent directory and review the configuration before running.
The packaged `subapplications.json` starts empty; use `subapplications.example.json`
as a reference for configuring applications. From an elevated terminal in that
directory, `./MrG.Daemon.exe install` installs the service and
`./MrG.Daemon.exe start` starts it. `stop` and `remove` undo those operations.
Running the EXE without arguments starts the daemon interactively.

CI verifies Go modules, runs vet/test compilation checks and builds the EXE.
It does not install or start the service; there are currently no Go unit tests.
The service uses legacy Windows utilities such as WMIC at runtime, which may need
separate installation on recent Windows versions.
