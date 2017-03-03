package calcium

import (
	"sync"

	enginetypes "github.com/docker/docker/api/types"
	"gitlab.ricebook.net/platform/core/types"
	"golang.org/x/net/context"
)

// remove images
func (c *calcium) RemoveImage(podname, nodename string, images []string) (chan *types.RemoveImageMessage, error) {
	ch := make(chan *types.RemoveImageMessage)

	node, err := c.GetNode(podname, nodename)
	if err != nil {
		return ch, err
	}

	opts := enginetypes.ImageRemoveOptions{
		Force:         false,
		PruneChildren: true,
	}

	go func() {
		wg := sync.WaitGroup{}
		wg.Add(len(images))

		for _, image := range images {
			go func(image string) {
				defer wg.Done()

				messages := []string{}
				success := true

				ms, err := node.Engine.ImageRemove(context.Background(), image, opts)
				if err != nil {
					success = false
					messages = append(messages, err.Error())
				} else {
					for _, m := range ms {
						if m.Untagged != "" {
							messages = append(messages, "Untagged: "+m.Untagged)
						}
						if m.Deleted != "" {
							messages = append(messages, "Deleted: "+m.Deleted)
						}
					}
				}
				ch <- &types.RemoveImageMessage{
					Image:    image,
					Success:  success,
					Messages: messages,
				}
			}(image)
		}

		wg.Wait()
		close(ch)
	}()

	return ch, nil
}
