package utils

import (
	"io"
	"mime/multipart"
)

func SplitByNReaders(file multipart.File, n int64) ([]io.Reader, error) {
    size, err := io.Copy(io.Discard, file)
    if err != nil {
        return nil, err
    }

    if size == 0 {
        return nil, nil
    }

    if size < n {
        return []io.Reader{file}, nil
    }

    if _, err = file.Seek(0, io.SeekStart); err != nil {
        return nil, err
    }

    nth := size/n
    readers := []io.Reader{}
    for i := int64(0); i < n-1; i++ {
        readers = append(readers, io.NewSectionReader(file, nth*i, nth))

        if _, err := file.Seek(nth*i, io.SeekCurrent); err != nil {
            return nil, err
        }
    }
    readers = append(readers, io.NewSectionReader(file, nth*(n-1), size-nth*(n-1)))

    return readers, nil
}