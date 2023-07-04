package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/hugolgst/rich-go/client"
)

// TODO use getMusicState and isMusicAppRunning to prevent crashing when music is paused, and to prevent unintentional booting of music app

func main() {
	lastAlbum := ""

	err := client.Login(os.Getenv("DISCORD_APP_ID"))
	if err != nil {
		panic(err)
	}

	for {
		songTitle, albumTitle, artistName := getSongMetaData()

		if albumTitle != lastAlbum {
			// TODO get album art and upload
		}

		lastAlbum = albumTitle

		songEndTime := getSongEndTime()
		currenTime := time.Now()

		err := client.SetActivity(client.Activity{
			State:      artistName,
			Details:    songTitle,
			LargeImage: "https://ucarecdn.com/7d12fc80-6bf6-4140-98b5-ff7c2cf91771/",
			LargeText:  albumTitle,
			SmallImage: "https://ucarecdn.com/c0a68a36-9de8-4572-816a-63093ae82690/",
			SmallText:  "And this is the small image",
			Timestamps: &client.Timestamps{
				Start: &currenTime,
				End:   &songEndTime,
			},
			Buttons: []*client.Button{
				{
					Label: "GitHub",
					Url:   "https://github.com/hugolgst/rich-go",
				},
			},
		})

		if err != nil {
			panic(err)
		}

		time.Sleep(5 * time.Second)
	}
}

func getMusicState() (string, error) {
	cmd := exec.Command("osascript", "-e", `tell application "Music" to player state as string`)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	state := strings.TrimSpace(string(output))
	return state, nil
}

func isMusicAppRunning() bool {
	cmd := exec.Command("osascript", "-e", `tell application "System Events" to (name of processes) contains "Music"`)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) == "true"
}

func uploadNewAlbumArt() { // TODO figure out way to resize images to reduce overall bandwidth usage when client downloads
	script := `tell application "Music"
					try
						if player state is not stopped then
							set alb to (get album of current track)
							tell artwork 1 of current track
								if format is JPEG picture then
									set imgFormat to ".jpg"
								else
									set imgFormat to ".png"
								end if
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
				set newPath to ((current_path as text) & "tmp" & imgFormat) as text
				--set newPath to ("Macintosh HD:Users:ollee:Programming:tmp" & imgFormat) as text
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
				end try`

	cmd := exec.Command("osascript", "-e", script)

	// Run the command and capture the output
	_, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
}

func getSongMetaData() (songTitle string, albumTitle string, artistName string) {
	// AppleScript code to get the song title, album title, and artist
	script := `tell application "Music"
					if player state is playing then
						set currentTrack to current track
						set songTitle to name of currentTrack
						set albumTitle to album of currentTrack
						set artistName to artist of currentTrack
						return {songTitle, albumTitle, artistName}
					end if
			   end tell`

	// Command to execute the AppleScript code using osascript
	cmd := exec.Command("osascript", "-e", script)

	// Run the command and capture the output
	output, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}

	// Remove leading/trailing whitespace and newline characters
	result := strings.TrimSpace(string(output))

	// Split the output into individual fields
	fields := strings.Split(result, ", ")

	// Extract the song title, album title, and artist
	songTitle = strings.Trim(fields[0], "\"")
	albumTitle = strings.Trim(fields[1], "\"")
	artistName = strings.Trim(fields[2], "\"")

	return
}

func getSongEndTime() time.Time {
	script := `tell application "Music"
		if player state is playing then
			return (duration of current track - player position)
		end if
	end tell`

	cmd := exec.Command("osascript", "-e", script)

	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error:", err)
		return time.Now()
	}

	now := time.Now()

	timeRemaining := string(output)

	timeRemainingList := strings.Split(timeRemaining, ".")

	myInt, err := strconv.Atoi(timeRemainingList[0])

	if err != nil {
		fmt.Println("Error:", err)
		return time.Now()
	}

	endTime := now.Add(time.Second * time.Duration(myInt))
	return endTime
}
