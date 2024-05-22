// Package obs creates elements in obs for the bot to use
package obs

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/andreykaipov/goobs"
	"github.com/andreykaipov/goobs/api/requests/inputs"
	"github.com/andreykaipov/goobs/api/requests/sceneitems"
	"github.com/andreykaipov/goobs/api/typedefs"
)

const (
	serverURL = "192.168.1.206:4455"
	soundPath = "/twitch/sound"
	gifPath   = "/twitch/gifs"
)

// Generate generates a random sound or gif to play in obs
func Generate(rewardType string) {
	client, err := goobs.New(serverURL)
	if err != nil {
		fmt.Println(err)
	}
	defer client.Disconnect()

	// List all inputs
	// resp, _ := client.Inputs.GetInputList()
	// for _, i := range resp.Inputs {
	// 	fmt.Printf("%+v\n", i)
	// }

	// list all scenes items
	// params := sceneitems.NewGetSceneItemListParams().WithSceneName("Coding")
	// resp, err := client.SceneItems.GetSceneItemList(params)
	// for _, v := range resp.SceneItems {
	// 	fmt.Printf("index=%d, id=%d, type=%s, name=%s\n", v.SceneItemIndex, v.SceneItemID, v.SourceType, v.SourceName)
	// }

	//Get existing input settings
	// params := inputs.NewGetInputSettingsParams().WithInputName("test")
	// src, err := client.Inputs.GetInputSettings(params)
	// fmt.Println(src.InputSettings)
	// map[local_file:C:/Users/migue/Music/np.mp3 restart_on_activate:false]
	// map[file:C:/Users/migue/Pictures/gifs/invoker.gif]

	// Get existing tranform
	// params := sceneitems.NewGetSceneItemTransformParams().
	// 	WithSceneItemId(47).
	// 	WithSceneName("Coding")
	// resp, err := client.SceneItems.GetSceneItemTransform(params)
	// fmt.Printf("%+v\n", resp.SceneItemTransform)
	// &{Alignment:5 BoundsAlignment:0 BoundsHeight:0 BoundsType:OBS_BOUNDS_NONE BoundsWidth:0 CropBottom:0 CropLeft:0 CropRight:0 CropTop:0 Height:465 PositionX:1403 PositionY:213 Rotation:0 ScaleX:1 ScaleY:1 SourceHeight:465 SourceWidth:498 Width:498}

	// Get monitor settings
	// p := inputs.NewGetInputAudioMonitorTypeParams().WithInputName("test")
	// t, err := client.Inputs.GetInputAudioMonitorType(p)
	// if err != nil {
	// 	fmt.Println(err)
	// }
	// fmt.Println(t.MonitorType)
	switch rewardType {
	case "sound":
		soundFile, err := randomFiles(soundPath)
		if err != nil {
			fmt.Println(err)
		}
		createInput(client, "rewardSound", "ffmpeg_source", soundFile)

	case "gif":

		gifFile, err := randomFiles(gifPath)
		if err != nil {
			fmt.Println(err)
		}
		createInput(client, "rewardGif", "image_source", gifFile)
	}
}

func randomFiles(path string) (string, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return "", err
	}
	var fileList []string
	for _, file := range files {
		if !file.IsDir() {
			fileList = append(fileList, file.Name())
		}
	}
	rand.New(rand.NewSource(time.Now().UnixNano()))
	randomIndex := rand.Intn(len(fileList))
	randomFile := fileList[randomIndex]
	return randomFile, nil
}

func createInput(client *goobs.Client, name string, kind string, path string) {
	params := inputs.NewCreateInputParams().
		WithSceneName("Coding").
		WithInputName(name).
		WithInputKind(kind)
	resp, err := client.Inputs.CreateInput(params)
	if err != nil {
		fmt.Println(err)
	}

	switch kind {
	case "ffmpeg_source":
		// change the settings
		params2 := inputs.NewSetInputSettingsParams().
			WithInputName(name).
			WithInputUuid(resp.InputUuid).
			WithInputSettings(map[string]interface{}{"local_file": "C:/Users/migue/Music/" + path, "restart_on_activate": false})
		client.Inputs.SetInputSettings(params2)
		// set monitor
		setMonitor(client, name, resp.InputUuid)
	case "image_source":
		// change the settings
		params2 := inputs.NewSetInputSettingsParams().
			WithInputName(name).
			WithInputUuid(resp.InputUuid).
			WithInputSettings(map[string]interface{}{"file": "C:/Users/migue/Pictures/gifs/" + path})
		client.Inputs.SetInputSettings(params2)
		setTransform(client, resp.SceneItemId, resp.InputUuid)
	}

	time.Sleep(3 * time.Second)
	deleteInput(client, name, resp.InputUuid)
}

func deleteInput(client *goobs.Client, name string, uuid string) {
	client.Inputs.RemoveInput(inputs.NewRemoveInputParams().WithInputName(name).WithInputUuid(uuid))
}

func setMonitor(client *goobs.Client, name string, uuid string) {
	params := inputs.NewSetInputAudioMonitorTypeParams().
		WithInputName(name).
		WithInputUuid(uuid).
		WithMonitorType("OBS_MONITORING_TYPE_MONITOR_AND_OUTPUT")
	_, err := client.Inputs.SetInputAudioMonitorType(params)
	if err != nil {
		fmt.Println(err)
	}
}

func setTransform(client *goobs.Client, id int, uuid string) {
	t := &typedefs.SceneItemTransform{
		Alignment:       5,
		BoundsAlignment: 0,
		BoundsHeight:    1,
		BoundsType:      "OBS_BOUNDS_NONE",
		BoundsWidth:     1,
		CropBottom:      0,
		CropLeft:        0,
		CropRight:       0,
		CropTop:         0,
		Height:          465,
		PositionX:       1403,
		PositionY:       213,
		Rotation:        0,
		ScaleX:          1,
		ScaleY:          1,
		SourceHeight:    465,
		SourceWidth:     498,
		Width:           498,
	}
	params := sceneitems.NewSetSceneItemTransformParams().
		WithSceneItemId(id).
		WithSceneName("Coding").
		WithSceneUuid(uuid).
		WithSceneItemTransform(t)
	_, err := client.SceneItems.SetSceneItemTransform(params)
	if err != nil {
		fmt.Println(err)
	}
}
