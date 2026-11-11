package images

import (
	"context"
	"fmt"

	"github.com/yellowdogai/yellowdog-sdk-go/internal"
	"github.com/yellowdogai/yellowdog-sdk-go/model"
)

// imagesClientProxy makes every request the Images client sends. Each method backs the
// ImagesClient method of the same name with its first letter lowercased.
type imagesClientProxy struct {
	http *internal.HTTPClient
}

func newImagesClientProxy(httpClient *internal.HTTPClient) *imagesClientProxy {
	return &imagesClientProxy{http: httpClient}
}

const imagesServicePath = "/images"

// --- Families ---------------------------------------------------------------------------------

func (p *imagesClientProxy) addImageFamily(ctx context.Context, family *model.MachineImageFamily) (*model.MachineImageFamily, error) {
	var result model.MachineImageFamily
	if err := p.http.Post(ctx, imagesServicePath+"/families", family, &result); err != nil {
		return nil, fmt.Errorf("add image family: %w", err)
	}
	return &result, nil
}

func (p *imagesClientProxy) updateImageFamily(ctx context.Context, family *model.MachineImageFamily) (*model.MachineImageFamily, error) {
	var result model.MachineImageFamily
	if err := p.http.Put(ctx, imagesServicePath+"/families", family, &result); err != nil {
		return nil, fmt.Errorf("update image family: %w", err)
	}
	return &result, nil
}

func (p *imagesClientProxy) deleteImageFamily(ctx context.Context, family *model.MachineImageFamily) error {
	path := fmt.Sprintf("%s/families/%s", imagesServicePath, family.Id.Or(""))
	if err := p.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("delete image family: %w", err)
	}
	return nil
}

func (p *imagesClientProxy) getImageFamily(ctx context.Context, familyId string) (*model.MachineImageFamily, error) {
	var result model.MachineImageFamily
	path := fmt.Sprintf("%s/families/%s", imagesServicePath, familyId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get image family: %w", err)
	}
	return &result, nil
}

func (p *imagesClientProxy) getImageFamilyByName(ctx context.Context, namespace string, familyName string) (*model.MachineImageFamily, error) {
	var result model.MachineImageFamily
	path := fmt.Sprintf("%s/namespaces/%s/families/%s", imagesServicePath, namespace, familyName)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get image family by name: %w", err)
	}
	return &result, nil
}

func (p *imagesClientProxy) getImageFamilies(search *MachineImageFamilySearch) *internal.SearchClient[MachineImageFamilySummary] {
	return internal.NewSearchClient[MachineImageFamilySummary](p.http, imagesServicePath+"/families", search)
}

// --- Groups -----------------------------------------------------------------------------------

func (p *imagesClientProxy) addImageGroup(ctx context.Context, family *model.MachineImageFamily, group *model.MachineImageGroup) (*model.MachineImageGroup, error) {
	var result model.MachineImageGroup
	path := fmt.Sprintf("%s/families/%s/groups", imagesServicePath, family.Id.Or(""))
	if err := p.http.Post(ctx, path, group, &result); err != nil {
		return nil, fmt.Errorf("add image group: %w", err)
	}
	return &result, nil
}

func (p *imagesClientProxy) updateImageGroup(ctx context.Context, group *model.MachineImageGroup) (*model.MachineImageGroup, error) {
	var result model.MachineImageGroup
	if err := p.http.Put(ctx, imagesServicePath+"/groups", group, &result); err != nil {
		return nil, fmt.Errorf("update image group: %w", err)
	}
	return &result, nil
}

func (p *imagesClientProxy) deleteImageGroup(ctx context.Context, group *model.MachineImageGroup) error {
	path := fmt.Sprintf("%s/groups/%s", imagesServicePath, group.Id.Or(""))
	if err := p.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("delete image group: %w", err)
	}
	return nil
}

func (p *imagesClientProxy) getImageGroup(ctx context.Context, groupId string) (*model.MachineImageGroup, error) {
	var result model.MachineImageGroup
	path := fmt.Sprintf("%s/groups/%s", imagesServicePath, groupId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get image group: %w", err)
	}
	return &result, nil
}

func (p *imagesClientProxy) getImageGroupByName(ctx context.Context, namespace string, familyName string, groupName string) (*model.MachineImageGroup, error) {
	var result model.MachineImageGroup
	path := fmt.Sprintf("%s/namespaces/%s/families/%s/groups/%s", imagesServicePath, namespace, familyName, groupName)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get image group by name: %w", err)
	}
	return &result, nil
}

func (p *imagesClientProxy) getLatestImageGroupByFamily(ctx context.Context, familyId string) (*model.MachineImageGroup, error) {
	var result model.MachineImageGroup
	path := fmt.Sprintf("%s/families/%s/groups/latest", imagesServicePath, familyId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get latest image group by family: %w", err)
	}
	return &result, nil
}

func (p *imagesClientProxy) getLatestImageGroupByFamilyByName(ctx context.Context, namespace string, familyName string) (*model.MachineImageGroup, error) {
	var result model.MachineImageGroup
	path := fmt.Sprintf("%s/namespaces/%s/families/%s/groups/latest", imagesServicePath, namespace, familyName)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get latest image group by family by name: %w", err)
	}
	return &result, nil
}

// --- Images -----------------------------------------------------------------------------------

func (p *imagesClientProxy) addImage(ctx context.Context, group *model.MachineImageGroup, image *model.MachineImage) (*model.MachineImage, error) {
	var result model.MachineImage
	path := fmt.Sprintf("%s/groups/%s/images", imagesServicePath, group.Id.Or(""))
	if err := p.http.Post(ctx, path, image, &result); err != nil {
		return nil, fmt.Errorf("add image: %w", err)
	}
	return &result, nil
}

func (p *imagesClientProxy) updateImage(ctx context.Context, image *model.MachineImage) (*model.MachineImage, error) {
	var result model.MachineImage
	if err := p.http.Put(ctx, imagesServicePath+"/images", image, &result); err != nil {
		return nil, fmt.Errorf("update image: %w", err)
	}
	return &result, nil
}

func (p *imagesClientProxy) deleteImage(ctx context.Context, image *model.MachineImage) error {
	path := fmt.Sprintf("%s/images/%s", imagesServicePath, image.Id.Or(""))
	if err := p.http.Delete(ctx, path); err != nil {
		return fmt.Errorf("delete image: %w", err)
	}
	return nil
}

func (p *imagesClientProxy) getImage(ctx context.Context, imageId string) (*model.MachineImage, error) {
	var result model.MachineImage
	path := fmt.Sprintf("%s/images/%s", imagesServicePath, imageId)
	if err := p.http.Get(ctx, path, &result); err != nil {
		return nil, fmt.Errorf("get image: %w", err)
	}
	return &result, nil
}
