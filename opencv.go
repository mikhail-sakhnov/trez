package trez

//#cgo CFLAGS: -Wall -Wextra -Os -Wno-unused-function -Wno-unused-parameter
//#cgo linux  pkg-config: opencv
//#cgo darwin pkg-config: opencv
//
//#include <opencv2/core/fast_math.hpp>
//#include <opencv/highgui.h>
//#include <opencv/cv.h>
//#include <opencv2/core/core_c.h>
//#include <opencv2/imgcodecs/imgcodecs_c.h>
//
//uchar* ptr_from_mat(CvMat* mat){
//	return mat->data.ptr;
//}
//
//void set_data_mat(CvMat* mat, void* ptr) {
//	mat->data.ptr = ptr;
//}
import "C"
import (
	"errors"
	"math"
	"unsafe"
)

var (
	errNoData              = errors.New("image data length is zero")
	errInvalidSourceFormat = errors.New("invalid data source format")
	errEncoding            = errors.New("error during encoding")
)

func Resize(data []byte, options Options) (*ProcessResult, error) {
	if len(data) == 0 {
		return nil, errNoData
	}

	// enable optimizations
	C.cvUseOptimized(1)

	// create a mat
	mat := C.cvCreateMat(1, C.int(len(data)), C.CV_8UC1)
	C.set_data_mat(mat, unsafe.Pointer(&data[0]))

	// Decode the source image
	src := C.cvDecodeImage(mat, C.CV_LOAD_IMAGE_UNCHANGED)
	C.cvReleaseMat(&mat)

	return resize(src, options)
}

func calcNewSize(options Options) (int, int) {
	width, height := options.Width, options.Height
	if options.MaxSide > 0 {
		maxSide := options.MaxSide
		if width <= maxSide && height <= maxSide {
			return width, height
		}
		var ratio float32
		if width >= height {
			ratio = float32(maxSide) / float32(width)
		} else {
			ratio = float32(maxSide) / float32(height)
		}
		width = int(float32(width) * ratio)
		height = int(float32(height) * ratio)
		return width, height
	}
	if options.MaxHeight > 0 {
		maxHeight := options.MaxHeight
		if height <= maxHeight {
			return width, height
		}
		ratio := float32(maxHeight) / float32(height)
		width = int(float32(width) * ratio)
		height = int(float32(height) * ratio)
		return width, height
	}
	if options.MaxWidth > 0 {
		maxWidth := options.MaxWidth
		if width <= maxWidth {
			return width, height
		}
		ratio := float32(maxWidth) / float32(width)
		width = int(float32(width) * ratio)
		height = int(float32(height) * ratio)
		return width, height
	}

	return width, height
}

func resize(src *C.IplImage, options Options) (*ProcessResult, error) {
	// Validate the source

	if src == nil || src.width == 0 || src.height == 0 {
		return nil, errInvalidSourceFormat
	}
	// Ensure the source will be freed.
	defer C.cvReleaseImage(&src)

	// Ensure options has Width and Height set.
	if options.Width == 0 {
		options.Width = int(src.width)
	}
	if options.Height == 0 {
		options.Height = int(src.height)
	}
	options.Width, options.Height = calcNewSize(options)

	// Check quality range
	if options.Quality < 0 {
		options.Quality = 0
	}
	if options.Quality > 100 {
		options.Quality = 100
	}
	if options.Quality == 0 {
		options.Quality = 85	// default value
	}

	// Get the size of the desired output image
	size := C.cvSize(C.int(options.Width), C.int(options.Height))

	// Get the x and y factors
	xf := float64(size.width) / float64(src.width)
	yf := float64(size.height) / float64(src.height)

	// Pointer to the final destination image.
	var dst *C.IplImage
	result := &ProcessResult{Width: options.Width, Height: options.Height}
	switch options.Algo {
	case FIT:
		ratio := math.Min(xf, yf)

		// Determine proper ROI rectangle placement
		rect := C.CvRect{}
		rect.width = C.int(math.Floor(float64(src.width) * ratio))
		rect.height = C.int(math.Floor(float64(src.height) * ratio))
		switch options.Gravity {
		case CENTER:
			rect.x = (size.width - rect.width) / 2
			rect.y = (size.height - rect.height) / 2
		case NORTH:
			rect.x = (size.width - rect.width) / 2
			rect.y = 0
		case NORTH_WEST:
			rect.x = 0
			rect.y = 0
		case NORTH_EAST:
			rect.x = (size.width - rect.width)
			rect.y = 0
		case SOUTH:
			rect.x = (size.width - rect.width) / 2
			rect.y = (size.height - rect.height)
		case SOUTH_WEST:
			rect.x = 0
			rect.y = (size.height - rect.height)
		case SOUTH_EAST:
			rect.x = (size.width - rect.width)
			rect.y = (size.height - rect.height)
		case WEST:
			rect.x = 0
			rect.y = (size.height - rect.height) / 2
		case EAST:
			rect.x = (size.width - rect.width)
			rect.y = (size.height - rect.height) / 2
		}

		// Initialize the output image
		dst = C.cvCreateImage(size, src.depth, src.nChannels)
		defer C.cvReleaseImage(&dst)

		b, g, r := options.Background[2], options.Background[1], options.Background[0]
		C.cvSet(unsafe.Pointer(dst), C.cvScalar(C.double(b), C.double(g), C.double(r), 0), nil)
		C.cvSetImageROI(dst, rect)
		C.cvResize(unsafe.Pointer(src), unsafe.Pointer(dst), C.CV_INTER_AREA)
		C.cvResetImageROI(dst)
	case FILL:
		// Algo: Scale image down keeping aspect ratio
		// constant, and then crop to requested size.
		ratio := math.Max(xf, yf)
		// Create an intermediate image
		intermediateSize := C.cvSize(
			C.int(math.Ceil(float64(src.width)*ratio)),
			C.int(math.Ceil(float64(src.height)*ratio)),
		)
		mid := C.cvCreateImage(intermediateSize, src.depth, src.nChannels)
		defer C.cvReleaseImage(&mid)

		C.cvResize(unsafe.Pointer(src), unsafe.Pointer(mid), C.CV_INTER_AREA)

		// Determine proper ROI rectangle placement
		rect := C.CvRect{}
		rect.width = size.width
		rect.height = size.height
		switch options.Gravity {
		case CENTER:
			rect.x = (mid.width - size.width) / 2
			rect.y = (mid.height - size.height) / 2
		case NORTH:
			rect.x = (mid.width - size.width) / 2
			rect.y = 0
		case NORTH_WEST:
			rect.x = 0
			rect.y = 0
		case NORTH_EAST:
			rect.x = (mid.width - size.width)
			rect.y = 0
		case SOUTH:
			rect.x = (mid.width - size.width) / 2
			rect.y = (mid.height - size.height)
		case SOUTH_WEST:
			rect.x = 0
			rect.y = (mid.height - size.height)
		case SOUTH_EAST:
			rect.x = (mid.width - size.width)
			rect.y = (mid.height - size.height)
		case WEST:
			rect.x = 0
			rect.y = (mid.height - size.height) / 2
		case EAST:
			rect.x = (mid.width - size.width)
			rect.y = (mid.height - size.height) / 2
		}

		C.cvSetImageROI(mid, rect)
		dst = (*C.IplImage)(C.cvClone(unsafe.Pointer(mid)))
		defer C.cvReleaseImage(&dst)
		C.cvResetImageROI(mid)
	}

	var params [6]C.int
	var ext *C.char
	switch options.Format {
	case JPEG:
		ext = C.CString(".jpg")
		params = [6]C.int{
			C.CV_IMWRITE_JPEG_QUALITY,
			C.int(options.Quality),	// from 0 to 100 (the higher is the better). Default value is 95.
			0,
			0,
			0,
			0,
		}
		if options.Progressive {
			params[3] = C.CV_IMWRITE_JPEG_PROGRESSIVE
		}
	case WEBP:
		ext = C.CString(".webp")
		// from 1 to 100 (the higher is the better).
		// By default (without any parameter) and for quality above 100 the lossless compression is used.
		q := options.Quality
		if q == 0 {
			q = 1
		}
		params = [6]C.int{
			C.CV_IMWRITE_WEBP_QUALITY,
			C.int(q),
			0,
			0,
			0,
			0,
		}
	case PNG:
		ext = C.CString(".png")
		// from 0 to 9. A higher value means a smaller size and longer compression time. Default value is 3.
		q := (100 - options.Quality) / 10
		if q > 9 {
			q = 9
		}
		params = [6]C.int{
			C.CV_IMWRITE_PNG_COMPRESSION,
			C.int(q),
			0,
			0,
			0,
			0,
		}
	}
	// encode
	ret := C.cvEncodeImage(ext, unsafe.Pointer(dst), &params[0])
	C.free(unsafe.Pointer(ext))

	if ret == nil {
		return nil, errEncoding
	}

	ptr := C.ptr_from_mat(ret)
	data := C.GoBytes(unsafe.Pointer(ptr), ret.step)
	C.cvReleaseMat(&ret)
	result.Data = data
	return result, nil
}

type ratio struct {
	src float64
	max float64
}
