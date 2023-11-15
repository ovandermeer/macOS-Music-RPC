# macOS-music-rpc
 A Discord Rich Presence client for Apple Music and the Music app on macOS

## Features
- Display song name, artist name, and album name
- Accurate album art pulled directly from the Music app
- Album art on custom songs imported to Music that aren't on Apple Music
- Low resource usage

## Installing
- Download the latest release from the [releases tab](https://github.com/ovandermeer/macOS-Music-RPC/releases)
- Run the following commands to start the application:
```
chmod +x macOS-music-rpc-arm64 // Make the file executable
xattr -d com.apple.quarantine // Disable macos security to allow the file to be run
./macOS-music-rpc-arm64 // Run the executable
```

## Compilation from souce
- Install the [Go programming language](https://go.dev/doc/install)
- Create an account with [UploadCare CDN](https://uploadcare.com/)
- Copy your UploadCare public and secret keys into `credentials.go`
- Run the following command to install the project dependencies:
```
go mod tidy
```
- Compile the executable:
```
go build .
```
- Run the executable:
```
./macOS-music-rpc
```