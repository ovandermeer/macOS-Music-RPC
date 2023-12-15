package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/hugolgst/rich-go/client"

	"github.com/uploadcare/uploadcare-go/file"
	"github.com/uploadcare/uploadcare-go/ucare"
	"github.com/uploadcare/uploadcare-go/upload"
)

const refreshSeconds = 5

type SongData struct {
	SongTitle string
	AlbumTitle string
	ArtistName string
}

func main() {
	lastSong := ""
	lastAlbum := ""
	lastArtist := ""

	lastTimeRemaining := 0

	albumArtURL := ""
	
	discordLoggedIn := false

	// Login to UploadCare API
	creds := ucare.APICreds{
		SecretKey: uploadCareSecretKey,
		PublicKey: uploadCarePublicKey,
	}

	conf := &ucare.Config{
		SignBasedAuthentication: true,
		APIVersion:              ucare.APIv06,
	}

	uCareClient, err := ucare.NewClient(creds, conf)
	if err != nil {
		panic(err)
	}

	// Main loop
	for {
		// Make sure music is playing
		if isMusicAppRunning() && getMusicState() == "playing" {
			songMetaData := getSongMetaData()

			// If music stopped playing between first check and now, then stop
			if songMetaData.SongTitle != "" && songMetaData.AlbumTitle != "" && songMetaData.ArtistName != "" {

				// If not currently logged into Discord RPC, attempt to login
				if !discordLoggedIn {
					client.Login("1111418320843976784")
					if err != nil {
						panic(err)
					}
					discordLoggedIn = true
				}

				songStartTime, songEndTime, timeRemaining := getSongTimestamps()

				// Checks if it's possible a status update may be required. If the song title/artist/album changes, or if the difference in time is greater than the refresh time
				// An extra second is added to the refresh time to account for rounding errors
				if math.Abs(float64(timeRemaining - lastTimeRemaining)) > refreshSeconds + 1 || songMetaData.SongTitle != lastSong || songMetaData.AlbumTitle != lastAlbum || songMetaData.ArtistName != lastArtist {
					
					// If the album has changed, check if the album art exists on CDN, otherwise upload new album art
					if songMetaData.AlbumTitle != lastAlbum {
						fileTitle := songMetaData.ArtistName + "-" + songMetaData.AlbumTitle + ".jpg"
						possibleAlbumArt := checkIfFileExists(fileTitle, uCareClient)
						if possibleAlbumArt != "" {
							albumArtURL = possibleAlbumArt
						} else {
							albumArtURL = uploadNewAlbumArt(fileTitle, uCareClient)
						}

						lastAlbum = songMetaData.AlbumTitle
					}

					// Set Discord activity
					err := client.SetActivity(client.Activity{
						State:      "by " + songMetaData.ArtistName,
						Details:    songMetaData.SongTitle,
						LargeImage: albumArtURL,
						LargeText:  songMetaData.AlbumTitle,
						Timestamps: &client.Timestamps{
							Start: &songStartTime,
							End:   &songEndTime,
						},
						Buttons: []*client.Button{
							{
								Label: "View on GitHub",
								Url:   "https://github.com/zvandermeer/macOS-Music-RPC",
							},
						},
					})

					if err != nil {
						panic(err)
					}
				}
				lastTimeRemaining = timeRemaining
			} else {
				// If music isn't playing, set status to blank and log out
				if discordLoggedIn {
					client.Logout()
					discordLoggedIn = false
				}
			}
		} else { 
			// If music isn't playing, set status to blank and log out
			if discordLoggedIn {
				client.Logout()
				discordLoggedIn = false
			}
		}

		time.Sleep(refreshSeconds * time.Second)
	}
}

// Check if music is playing
func getMusicState() string {
	cmd := exec.Command("osascript", "-e", `tell application "Music" to player state as string`)
	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	state := strings.TrimSpace(string(output))
	return state
}

// Checks to make sure the music app is running before making any requests, to make sure the app isn't unintentionally launched
func isMusicAppRunning() bool {
	cmd := exec.Command("osascript", "-e", `tell application "System Events" to (name of processes) contains "Music"`)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) == "true"
}

// Uploads the album art of the currently playing song to the CDN
func uploadNewAlbumArt(fileTitle string, uCareClient ucare.Client) string {
	path, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	// Replaces all ":" characters with the "/" character. AppleScript uses ":" to deliminate file paths, 
	// but for some reason changes "/" characters into ":" characters when writing to a file.
	fileTitle = strings.ReplaceAll(fileTitle, ":", "/")

	// Change all "/" characters to ":" characters for AppleScript file paths
	appleScriptPath := "Macintosh HD" + path
	appleScriptPath = strings.ReplaceAll(appleScriptPath, "/", ":")
	appleScriptPath += ":" + fileTitle

	// Because AppleScript changes all "/" characters to ":" characters, we have to do the same here
	fileTitle = strings.ReplaceAll(fileTitle, "/", ":")

	script := fmt.Sprintf(`tell application "Music"
					try
						if player state is not stopped then
							set alb to (get album of current track)
							tell artwork 1 of current track
							end tell
							set rawData to (get raw data of artwork 1 of current track)
						else
							return
						end if
					on error
						return
					end try
				end tell
				--get current path
				tell application "Finder"
					set current_path to container of (path to me) as alias
				end tell
				--create path to save image as jpg or png
				set newPath to ("%s") as text
				--display dialog (newPath as text) buttons {"OK"}
				try
					--create file
					tell me to set fileRef to (open for access newPath with write permission)
					--overwrite existing file
					write rawData to fileRef starting at 0
					tell me to close access fileRef
				on error m number n
					log n
					log m
					try
						tell me to close access fileRef
					end try
				end try`, appleScriptPath)

	cmd := exec.Command("osascript", "-e", script)

	_, err = cmd.Output()
	if err != nil {
		panic(err)
	}

	// Load the created file, upload it, then delete it to reduce clutter
	path += "/" + fileTitle

	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}

	uploadSvc := upload.NewService(uCareClient)

	params := upload.FileParams{
		Data:        f,
		Name:        f.Name(),
		ContentType: "image/jpg",
	}
	fID, err := uploadSvc.File(context.Background(), params)
	if err != nil {
		panic(err)
	}

	f.Close()

	os.Remove(path)

	return "https://ucarecdn.com/" + fID + "/-/preview/938x432/-/quality/smart/-/format/auto/"
}

// Checks if album art already exists on CDN to reduce upload usage
func checkIfFileExists(fileTitle string, uCareClient ucare.Client) string {
	fileSvc := file.NewService(uCareClient)

	listParams := file.ListParams{
		Stored:  ucare.Bool(true),
		OrderBy: ucare.String(file.OrderBySizeAsc),
	}

	fileList, err := fileSvc.List(context.Background(), listParams)
	if err != nil {
		panic(err)
	}

	for fileList.Next() {
		finfo, err := fileList.ReadResult()
		if err != nil {
			panic(err)
		}

		if finfo.BasicFileInfo.OriginalFileName == fileTitle {
			return "https://ucarecdn.com/" + finfo.ID + "/-/preview/938x432/-/quality/smart/-/format/auto/"
		}
	}
	return ""
}

// Gets metadata about the currently playing song
func getSongMetaData() (myMetadata SongData) {
	// Applescript code to get metadata from the Music app
	script := `tell application "Music"
					if player state is playing then
						set currentTrack to current track
						set songTitle to name of currentTrack
						set albumTitle to album of currentTrack
						set artistName to artist of currentTrack
						return "{\"songTitle\":\"" & songTitle & "\",\"albumTitle\":\"" & albumTitle & "\",\"artistName\":\"" & artistName & "\"}"				
					end if
				end tell`

	cmd := exec.Command("osascript", "-e", script)

	output, err := cmd.Output()
	if err != nil {
		panic(err)
	}

	// Remove leading/trailing whitespace and newline characters
	result := strings.TrimSpace(string(output))

	json.Unmarshal([]byte(result), &myMetadata)

	return 
}

// Returns the start and end time of the song
func getSongTimestamps() (time.Time, time.Time, int) {
	script := `tell application "Music"
		if player state is playing then
			return {duration of current track, player position}
		end if
	end tell`

	cmd := exec.Command("osascript", "-e", script)

	output, err := cmd.Output()
	if err != nil {
		return time.Now(), time.Now(), 0
	}

	// Remove leading/trailing whitespace and newline characters
	result := strings.TrimSpace(string(output))

	fields := strings.Split(result, ", ")

	now := time.Now()

	// Calculate timestamps
	songDuration, err := strconv.Atoi(strings.Split(fields[0], ".")[0])
	if err != nil {
		return time.Now(), time.Now(), 0
	}
	timeElapsed, err := strconv.Atoi(strings.Split(fields[1], ".")[0])
	if err != nil {
		return time.Now(), time.Now(), 0
	}

	timeRemaining := songDuration - timeElapsed

	endTime := now.Add(time.Second * time.Duration(timeRemaining))
	startTime := now.Add(-(time.Second * time.Duration(songDuration - timeRemaining)))

	return startTime, endTime, timeRemaining
}
