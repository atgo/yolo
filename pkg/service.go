package pkg

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func (this Lock) Install() error {
	eg := errgroup.Group{}

	for _, p := range this.Packages {
		pkg := p

		eg.Go(
			func() error {
				return this.installPackage(pkg)
			},
		)
	}

	return eg.Wait()
}

func (this Lock) installPackage(pkg Package) error {
	if "" == pkg.Dist.Url {
		logrus.
			WithField("name", pkg.Name).
			Errorln("invalid distribution source")

		return fmt.Errorf("invalid distribution source")
	}

	if "zip" != pkg.Dist.Type {
		logrus.
			WithField("name", pkg.Name).
			WithField("dist.type", pkg.Dist.Type).
			Errorln("unsupported source type")

		return fmt.Errorf("unsupported source type")
	} else {
		name := strings.Replace(pkg.Name, "/", "-", 1)
		zipPath := "vendor/" + name + ".zip"
		if err := this.downloadZipFile(pkg, zipPath); nil != err {
			return errors.Wrap(err, "download zip file")
		}

		if err := this.unzip(zipPath, "vendor/"+pkg.Name); nil != err {
			return errors.Wrap(err, "unzip "+zipPath)
		} else if err := os.Remove(zipPath); nil != err {
			return errors.Wrap(err, "clean up")
		}
	}

	return nil
}

func (this Lock) downloadZipFile(pkg Package, target string) error {
	res, err := http.DefaultClient.Get(pkg.Dist.Url)
	if nil != err {
		return err
	} else {
		defer res.Body.Close()
	}

	file, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if nil != err {
		return err
	} else {
		defer file.Close()
	}

	logrus.
		WithField("dist.url", pkg.Dist.Url).
		WithField("name", pkg.Name).
		WithField("version", pkg.Version).
		Infoln("Downloading")

	if 200 != res.StatusCode {
		return fmt.Errorf("should see 200")
	}

	b, err := ioutil.ReadAll(res.Body)
	if nil != err {
		return err
	}

	_, err = file.Write(b)

	return err
}

func (this Lock) unzip(src string, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}

	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	os.MkdirAll(dest, 0755)

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		paths := strings.Split(f.Name, "/")
		paths[0] = dest
		path := filepath.Join(paths...)

		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}
