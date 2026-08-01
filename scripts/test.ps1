$ErrorActionPreference = 'Stop'

npm --prefix web test -- --run
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
npm --prefix web run build
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
go test ./... -count=1
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
go vet ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
npm --prefix web run e2e
exit $LASTEXITCODE
