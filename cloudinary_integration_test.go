// +build integration

package cloudinary_test

import (
	"os"
	"testing"

	"log"

	cloudinary "github.com/alioygur/cloudinary-go"
)

func TestMain(t *testing.M) {
	// check env variables
	if os.Getenv("CLOUDINARY_URL") == "" {
		log.Fatal("you must set the env: CLOUDINARY_URL to run integration tests")
	}

	os.Exit(t.Run())
}

func TestCloudinary_Upload(t *testing.T) {
	var imagename = "testimage"
	c, err := cloudinary.New(os.Getenv("CLOUDINARY_URL"))
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open("./testdata/cloudinary.png")
	if err != nil {
		t.Fatalf("can't open test image: %v", err)
	}
	defer f.Close()

	img, err := c.Upload(f, imagename)
	if err != nil {
		t.Errorf("upload failed: %v", err)
	}

	if img.PublicID != imagename {
		t.Errorf("want public_id %s, got %s", imagename, img.PublicID)
	}

	// delete test image
	if err := c.Delete(imagename); err != nil {
		t.Errorf("image delete failed after upload: %v", err)
	}
}

func TestCloudinary_UploadVideo(t *testing.T) {
	var videoname = "testvideo"
	c, err := cloudinary.New(os.Getenv("CLOUDINARY_URL"))
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open("./testdata/cloudinary.ogv")
	if err != nil {
		t.Fatalf("can't open test video: %v", err)
	}
	defer f.Close()

	res, err := c.UploadVideo(f, videoname)
	if err != nil {
		t.Errorf("upload failed: %v", err)
		return
	}

	if res.PublicID != videoname {
		t.Errorf("want public_id %s, got %s", videoname, res.PublicID)
		return
	}

	// delete test video
	if err := c.DeleteVideo(videoname); err != nil {
		t.Errorf("video delete failed after upload: %v", err)
	}
}
