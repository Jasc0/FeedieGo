package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
	"image"
	_ "image/jpeg"
	_ "image/png"
)


type FeedieImageBackendProvider int
const(
	kitty FeedieImageBackendProvider = iota
	ueberzug
	none
)

var ImageMapMutex sync.Mutex

type thumbnailManager struct{
	url_to_path map[string] string
	current string
	showing bool
	enabled bool
	backend FeedieImageBackendProvider
	directory string
	devtty *os.File
	ueberzug *ueberzugClient
}

func initThumbnailManager(config FeedieConfig) thumbnailManager{
	if _, err := os.Stat(config.ThumbnailPath); os.IsNotExist(err) {
       // Directory does not exist, create it (with all parent directories)
       err := os.MkdirAll(config.ThumbnailPath, 0755)
       if err != nil {
           log.Fatal("Error creating directory:", err)
       }
   } else if err != nil {
       log.Fatal("Error checking directory:", err)
   } 	
	tm := thumbnailManager{
		url_to_path: make(map[string]string),
		current: "",
		showing: false,
		enabled: true,
		backend: config.getThumbnailBackend(),
		directory: config.ThumbnailPath,
	}

	devtty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil{ log.Fatal(err)}
	tm.devtty = devtty
	if err != nil{ log.Fatal(err)}


	switch(tm.backend){
	case ueberzug:
		tm.ueberzug = ueberzugStart(tm)
		tm.ueberzug.scaler = config.ThumbnailScaler
	}
	return tm
}

func (tm thumbnailManager) preloadImages(urls []string){
	ImageMapMutex.Lock()
	defer ImageMapMutex.Unlock()
	for _, u := range urls{
		_, ok := tm.url_to_path[u]
		if !ok{
			desired_path := fmt.Sprintf("%s/%s",tm.directory,GetHashString(u))
			_, err := os.Stat(desired_path)
			if err == nil {
				tm.url_to_path[u] = desired_path
				return
			}

			if download_file(u,desired_path){
				tm.url_to_path[u] = desired_path
			}
		}
		
	}
}

func (tm thumbnailManager) drawImage (x, y, width, height int, url string) bool{
	if !tm.enabled{
		return false
	}
	// download thumbnail and store it in map
	ImageMapMutex.Lock()
	path, ok := tm.url_to_path[url]
	ImageMapMutex.Unlock()
	if !ok{
		desired_path := fmt.Sprintf("%s/%s",tm.directory,GetHashString(url))
		_, err := os.Stat(desired_path)
		if err == nil {
			ImageMapMutex.Lock()
			tm.url_to_path[url] = desired_path
			ImageMapMutex.Unlock()
		} else{
			if download_file(url,desired_path){
				ImageMapMutex.Lock()
				tm.url_to_path[url] = desired_path
				ImageMapMutex.Unlock()
				path = desired_path
			}
		}
	}
	if path == "" {return false}

	// display image
	switch(tm.backend){
	case kitty:
		imgBytes, _ := os.ReadFile(path)
		cmd := exec.Command("kitten", "icat",  
		fmt.Sprintf("--place=%dx%d@%dx%d",width,height,x,y), 
		"--stdin=yes","--scale-up=yes", "--transfer-mode=stream", "--image-id=1")	
		cmd.Stdout =tm.devtty
		cmd.Stderr = tm.devtty
		cmd.Stdin  = bytes.NewReader(imgBytes)
		err := cmd.Run()
		if err != nil{
			return false
		}
		tm.current = url
		tm.showing = true
		return true
	case ueberzug:
			tm.ueberzug.Show(x , y, width, height, path)
		tm.current = url
		tm.showing = true
		return true
	}
	return false
}
func (tm thumbnailManager) clear (){
	tm.current =""
	tm.showing =false
	devtty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil{ log.Fatal(err)}
	defer devtty.Close()
	switch(tm.backend){
	case kitty:
		seq := fmt.Sprintf("\x1b_Ga=d,d=i,i=%d\x1b\\", 1)
    	devtty.WriteString(seq)
	case ueberzug:
		tm.ueberzug.Hide()
	}
}

func download_file (url string, filepath string) bool{
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	out, err := os.Create(filepath)
	if err != nil {
		return false
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return false
	}
	return true
}
func GetHashString(root string) string{
	hasher := fnv.New64a()
	hasher.Write([]byte(root))
	return fmt.Sprintf("%x", hasher.Sum64())
}
// ueberzug stuff

type ueberzugClient struct{
	cmd *exec.Cmd
	in io.WriteCloser
	mu sync.Mutex
	scaler string
}

func ueberzugStart(tm thumbnailManager) *ueberzugClient{
	cmd := exec.Command("ueberzug", "layer", "--parser", "json")
	// Attach to the current terminal. Do NOT point Stdout/Stderr to another TTY.
	cmd.Stdout = tm.devtty
	cmd.Stderr = tm.devtty

	w, err := cmd.StdinPipe()
	if err != nil { log.Fatal(err) }

	if err := cmd.Start(); err != nil {
		// Helpful env hints:
		if os.Getenv("DISPLAY") == "" {
			log.Println("Hint: DISPLAY is empty; ueberzug requires X11.")
		}
		log.Fatal(err)
	}
	return &ueberzugClient{cmd: cmd, in: w}
}

func (uzc *ueberzugClient)Show(x, y, width, height int, path string){
	var XOffset, YOffset int
	IAR, err := GetAspectRatio(path)
	if err != nil || in(uzc.scaler, []string{"distort","crop"}){
		XOffset, YOffset = 0, 0
	} else{
		CELL_R := CELL_W/CELL_H
		XOffset = max(0, width/2 - int(IAR*float64(height)*1/CELL_R)/2)
		YOffset = max(0,int(float64(height)/(2*CELL_R) - float64(width)/(2*IAR)))
	}
	msg := map[string]any{
		"action":     "add",
		"identifier": "1",
		"x":          x + XOffset,
		"y":          y + YOffset,
		"width":      width,
		"height":     height,
		"path":       path,
		"scaler": uzc.scaler,
	}
	bytes, err := json.Marshal(msg)
	if err != nil { log.Fatal(err) }
	bytes = append(bytes, '\n')
	uzc.mu.Lock()
	defer uzc.mu.Unlock()
	if _, err := uzc.in.Write(bytes); err != nil { log.Fatal(err) }
}

func (uzc *ueberzugClient)Hide(){
	msg := map[string]any{
		"action":     "remove",
		"identifier": "1",
	}
	bytes, err := json.Marshal(msg)
	if err != nil { log.Fatal(err) }
	bytes = append(bytes, '\n')
	uzc.mu.Lock()
	defer uzc.mu.Unlock()
	if _, err := uzc.in.Write(bytes); err != nil { log.Fatal(err) }
}



// GetAspectRatio takes an image file path and returns its aspect ratio (width / height).
func GetAspectRatio(imagePath string) (float64, error) {
	file, err := os.Open(imagePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, fmt.Errorf("failed to decode image: %w", err)
	}

	if img.Height == 0 {
		return 0, fmt.Errorf("image height is zero")
	}

	aspectRatio := float64(img.Width) / float64(img.Height)
	return aspectRatio, nil
}

