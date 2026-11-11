// Package images provides access to the YellowDog Machine Images API.
//
// Machine images are organised in a hierarchy:
//
//	MachineImageFamily
//	└── MachineImageGroup  (e.g. "ubuntu-22.04-amd64")
//	    └── MachineImage   (a specific image version/build)
//
// Use [ImagesClient] to create families and groups, add images, and find the
// latest group in a family:
//
//	family, _ := client.Images.AddImageFamily(ctx, &model.MachineImageFamily{...})
//	group, _ := client.Images.AddImageGroup(ctx, family, &model.MachineImageGroup{...})
//	image, _ := client.Images.AddImage(ctx, group, &model.MachineImage{...})
//	latest, _ := client.Images.GetLatestImageGroupByFamily(ctx, family.GetId())
package images
