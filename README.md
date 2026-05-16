# omnivideo-sdk-go

Go SDK for [Omni Video](https://omnivideo.net/) — generate video and image content with the **Gemini Omni Video** series of models.

[Omni Video](https://omnivideo.net/) hosts the Gemini Omni Video family (`seedance-2` for text/image → video, `gpt-image-2` and `nano-banana-2` for text/image → image) behind one simple REST API.

## Install

```bash
go get github.com/beamtrace/omnivideo-sdk-go
```

## Get an API key

Sign in at **<https://omnivideo.net/>**, open the account page, then create a `sk-…` token.

```bash
export OMNIVIDEO_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"

	omnivideo "github.com/beamtrace/omnivideo-sdk-go"
)

func main() {
	client, err := omnivideo.NewClient("") // reads OMNIVIDEO_API_KEY
	if err != nil {
		log.Fatal(err)
	}
	task, err := client.Run(
		context.Background(),
		omnivideo.CreateTaskInput{
			ModelID:     "seedance-2",
			Prompt:      "a serene zen garden at sunrise, ultra detailed",
			AspectRatio: "16:9",
		},
		omnivideo.RunOptions{},
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(task.OutputURL())
}
```

### Lower level: create + poll

```go
task, err := client.CreateTask(ctx, omnivideo.CreateTaskInput{
	ModelID: "gpt-image-2",
	Prompt:  "cyberpunk corgi, neon rim light",
})
// ...
task, err = client.GetTask(ctx, task.TaskID)
```

## Models

| `model_id`      | Modality           | Output      |
| --------------- | ------------------ | ----------- |
| `seedance-2`    | text/image → video | `video_url` |
| `gpt-image-2`   | text/image → image | `image_url` |
| `nano-banana-2` | text/image → image | `image_url` |

See the live model list on [omnivideo.net](https://omnivideo.net/).

## Links

- Website & account: <https://omnivideo.net/>
- API docs: <https://omnivideo.net/api-docs>

## License

MIT
