package gopro

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/karrick/godirwalk"
	"github.com/konradit/mmt/pkg/utils"
	"github.com/minio/minio/pkg/disk"
	"gopkg.in/djherbis/times.v1"
)

/*
Uses data from:
https://community.gopro.com/t5/en/GoPro-Camera-File-Naming-Convention/ta-p/390220#
*/

func Import(in, out, dateFormat string, bufferSize int, prefix string, dateRange []string) (*utils.Result, error) {
	gpVersion, err := readInfo(in)
	if err != nil {
		return nil, err
	}

	di, err := disk.GetInfo(in)
	if err != nil {
		return nil, err
	}
	percentage := (float64(di.Total-di.Free) / float64(di.Total)) * 100

	c := color.New(color.FgCyan, color.Bold)
	y := color.New(color.FgHiBlue, color.Bold)
	color.Cyan("🎥 [%s]:", gpVersion.CameraType)
	c.Printf("\t📹 FW: %s ", gpVersion.FirmwareVersion)
	y.Printf("SN: %s\n", gpVersion.CameraSerialNumber)
	color.Cyan("\t💾 %s/%s (%0.2f%%)\n",
		humanize.Bytes(di.Total-di.Free),
		humanize.Bytes(di.Total),
		percentage,
	)

	root := strings.Split(gpVersion.FirmwareVersion, ".")[0]

	if prefix == "cameraname" {
		prefix = gpVersion.CameraType
	}

	dateStart := time.Date(0000, time.Month(1), 1, 0, 0, 0, 0, time.UTC)

	dateEnd := time.Now()

	if len(dateRange) != 0 {
		if len(dateRange) == 1 {
			switch dateRange[0] {
			case "today":
				dateStart = time.Date(dateEnd.Year(), dateEnd.Month(), dateEnd.Day(), 0, 0, 0, 0, dateEnd.Location())
			case "yesterday":
				dateStart = time.Date(dateEnd.Year(), dateEnd.Month(), dateEnd.Day(), 0, 0, 0, 0, dateEnd.Location()).Add(-24 * time.Hour)
			case "week":
				dateStart = time.Date(dateEnd.Year(), dateEnd.Month(), dateEnd.Day(), 0, 0, 0, 0, dateEnd.Location()).Add(-24 * time.Duration((int(dateEnd.Weekday()) - 1)) * time.Hour)
			}
		}

		if len(dateRange) == 2 {
			replacer := strings.NewReplacer("day", "02", "month", "01", "year", "2006")
			start, err := time.Parse(replacer.Replace(replacer.Replace(dateFormat)), dateRange[0])
			if err == nil {
				dateStart = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
			}
			end, err := time.Parse(replacer.Replace(replacer.Replace(dateFormat)), dateRange[1])
			if err == nil {
				dateEnd = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
			}

		}
	}
	switch root {
	case "HD6", "HD7", "HD8", "HD9":
		result := importFromGoProV2(filepath.Join(in, fmt.Sprint(DCIM)), out, SortOptions{
			ByDays:             true,
			SkipAuxiliaryFiles: false,
			AddHiLightTags:     true,
			ByCamera:           true,
			DateFormat:         dateFormat,
			BufferSize:         bufferSize,
			Prefix:             prefix,
			DateRange:          []time.Time{dateStart, dateEnd},
		}, gpVersion.CameraType)
		return &result, nil
	case "HD2,", "HD3", "HD4", "HX", "HD5":
		result := importFromGoProV1(filepath.Join(in, fmt.Sprint(DCIM)), out, SortOptions{
			ByDays:             true,
			SkipAuxiliaryFiles: false,
			AddHiLightTags:     true,
			ByCamera:           true,
			DateFormat:         dateFormat,
			BufferSize:         bufferSize,
			Prefix:             prefix,
			DateRange:          []time.Time{dateStart, dateEnd},
		}, gpVersion.CameraType)
		return &result, nil
	case "H19":

		result := importFromMAX(filepath.Join(in, fmt.Sprint(DCIM)), out, SortOptions{
			ByDays:             true,
			SkipAuxiliaryFiles: false,
			AddHiLightTags:     true,
			ByCamera:           true,
			DateFormat:         dateFormat,
			BufferSize:         bufferSize,
			Prefix:             prefix,
			DateRange:          []time.Time{dateStart, dateEnd},
		})
		return &result, nil
	default:
		return nil, errors.New(fmt.Sprintf("Camera `%s` is not supported", gpVersion.CameraType))
	}
}

func importFromMAX(root string, output string, sortoptions SortOptions) utils.Result {
	mediaFolder := `\d\d\dGOPRO`

	fileTypes := []FileTypeMatch{
		{
			Regex:    regexp.MustCompile(`GS\d+.360`),
			Type:     Video,
			HeroMode: false,
		},
		{
			Regex:    regexp.MustCompile(`GS_+\d+.JPG`),
			Type:     Photo,
			HeroMode: false,
		},
		{
			Regex:    regexp.MustCompile(`GP_+\d+.JPG`),
			Type:     Photo,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`GH\d+.MP4`),
			Type:     Video,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`GPAA\d+.JPG`),
			Type:     Multishot,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`GH\d+.LRV`),
			Type:     LowResolutionVideo,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`GH\d+.THM`),
			Type:     Thumbnail,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`GS\d+.LRV`),
			Type:     LowResolutionVideo,
			HeroMode: false,
		},
		{
			Regex:    regexp.MustCompile(`GS\d+.THM`),
			Type:     Thumbnail,
			HeroMode: false,
		},
	}
	var result utils.Result
	/*
		The idea is to have a result like:

		20-02-2021/MAX/photos/
							normal/
						   			GS_0001.JPG
							powerpano/
						   			GP_0001.JPG
					   /videos
						   /heromode
									/GH012345.MP4
						   /360
								/GS012345.MP4

	*/
	folders, err := ioutil.ReadDir(root)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return result
	}

	for _, f := range folders {
		r, err := regexp.MatchString(mediaFolder, f.Name())
		if err != nil {
			result.Errors = append(result.Errors, err)
		}
		if r {
			color.Green("Looking at %s", f.Name())

			err = godirwalk.Walk(filepath.Join(root, f.Name()), &godirwalk.Options{
				Callback: func(osPathname string, de *godirwalk.Dirent) error {

					for _, ftype := range fileTypes {
						if ftype.Regex.MatchString(de.Name()) {

							if sortoptions.ByDays {
								t, err := times.Stat(osPathname)
								if err != nil {
									log.Fatal(err.Error())
								}
								if t.HasBirthTime() {
									d := t.BirthTime()
									mediaDate := d.Format("02-01-2006")
									if strings.Contains(sortoptions.DateFormat, "year") && strings.Contains(sortoptions.DateFormat, "month") && strings.Contains(sortoptions.DateFormat, "day") {
										replacer := strings.NewReplacer("day", "02", "month", "01", "year", "2006")
										mediaDate = d.Format(replacer.Replace(sortoptions.DateFormat))
									}

									dayFolder := filepath.Join(output, mediaDate)
									if _, err := os.Stat(dayFolder); os.IsNotExist(err) {
										os.Mkdir(dayFolder, 0755)
									}

									if sortoptions.ByCamera {
										if _, err := os.Stat(filepath.Join(dayFolder, "MAX")); os.IsNotExist(err) {
											os.Mkdir(filepath.Join(dayFolder, "MAX"), 0755)
										}
										dayFolder = filepath.Join(dayFolder, "MAX")
									}

									switch ftype.Type {
									case Video:
										x := de.Name()

										filename := fmt.Sprintf("%s%s-%s.%s", x[:2], x[4:][:4], x[2:][:2], strings.Split(x, ".")[1])
										color.Green(">>> %s", filename, color.Bold)

										foldersNeeded := []string{"videos/360", "videos/heromode"}
										for _, fn := range foldersNeeded {
											if _, err := os.Stat(filepath.Join(dayFolder, fn)); os.IsNotExist(err) {
												err = os.MkdirAll(filepath.Join(dayFolder, fn), 0755)
												if err != nil {
													log.Fatal(err.Error())
												}
											}
										}

										dest := foldersNeeded[1]
										if !ftype.HeroMode {
											dest = foldersNeeded[0]
										}
										err = utils.CopyFile(osPathname, filepath.Join(dayFolder, dest, filename), 1000)
										if err != nil {
											result.Errors = append(result.Errors, err)
											result.FilesNotImported = append(result.FilesNotImported, osPathname)
										} else {
											result.FilesImported += 1
										}
									case Photo:
										foldersNeeded := []string{"photos/360", "photos/heromode"}
										for _, fn := range foldersNeeded {
											if _, err := os.Stat(filepath.Join(dayFolder, fn)); os.IsNotExist(err) {
												err = os.MkdirAll(filepath.Join(dayFolder, fn), 0755)
												if err != nil {
													log.Fatal(err.Error())
												}
											}
										}

										dest := foldersNeeded[1]
										if !ftype.HeroMode {
											dest = foldersNeeded[0]
										}
										color.Green(">>> %s", de.Name(), color.Bold)

										err = utils.CopyFile(osPathname, filepath.Join(dayFolder, dest, de.Name()), 1000)
										if err != nil {
											result.Errors = append(result.Errors, err)
											result.FilesNotImported = append(result.FilesNotImported, osPathname)
										} else {
											result.FilesImported += 1
										}
									case PowerPano:
										foldersNeeded := []string{"photos/powerpano"}
										for _, fn := range foldersNeeded {
											if _, err := os.Stat(filepath.Join(dayFolder, fn)); os.IsNotExist(err) {
												err = os.MkdirAll(filepath.Join(dayFolder, fn), 0755)
												if err != nil {
													log.Fatal(err.Error())
												}
											}
										}

										dest := foldersNeeded[1]
										if !ftype.HeroMode {
											dest = foldersNeeded[0]
										}
										color.Green(">>> %s", de.Name(), color.Bold)

										err = utils.CopyFile(osPathname, filepath.Join(dayFolder, dest, de.Name()), 1000)
										if err != nil {
											result.Errors = append(result.Errors, err)
											result.FilesNotImported = append(result.FilesNotImported, osPathname)
										} else {
											result.FilesImported += 1
										}
									case LowResolutionVideo:
										if !sortoptions.SkipAuxiliaryFiles {
											foldersNeeded := []string{"videos/proxy/heromode", "videos/proxy/360"}
											for _, fn := range foldersNeeded {
												if _, err := os.Stat(filepath.Join(dayFolder, fn)); os.IsNotExist(err) {
													err = os.MkdirAll(filepath.Join(dayFolder, fn), 0755)
													if err != nil {
														log.Fatal(err.Error())
													}
												}
											}
											dest := foldersNeeded[1]
											if ftype.HeroMode {
												dest = foldersNeeded[0]
											}
											x := de.Name()

											filename := fmt.Sprintf("%s%s-%s.%s", x[:2], x[4:][:4], x[2:][:2], strings.Split(x, ".")[1])
											err = utils.CopyFile(osPathname, filepath.Join(dayFolder, dest, filename), 1000)
											if err != nil {
												result.Errors = append(result.Errors, err)
												result.FilesNotImported = append(result.FilesNotImported, osPathname)
											} else {
												result.FilesImported += 1
											}
										}
									case Thumbnail:
										if !sortoptions.SkipAuxiliaryFiles {
											foldersNeeded := []string{"videos/thumbnails/heromode", "videos/thumbnails/360"}
											for _, fn := range foldersNeeded {
												if _, err := os.Stat(filepath.Join(dayFolder, fn)); os.IsNotExist(err) {
													err = os.MkdirAll(filepath.Join(dayFolder, fn), 0755)
													if err != nil {
														log.Fatal(err.Error())
													}
												}
											}
											dest := foldersNeeded[1]
											if ftype.HeroMode {
												dest = foldersNeeded[0]
											}
											x := de.Name()

											filename := fmt.Sprintf("%s%s-%s.%s", x[:2], x[4:][:4], x[2:][:2], strings.Split(x, ".")[1])
											err = utils.CopyFile(osPathname, filepath.Join(dayFolder, dest, filename), 1000)
											if err != nil {
												result.Errors = append(result.Errors, err)
												result.FilesNotImported = append(result.FilesNotImported, osPathname)
											} else {
												result.FilesImported += 1
											}
										}
									default:
										color.Red("Unsupported file %s", de.Name())
										result.Errors = append(result.Errors, errors.New("Unsupported file "+de.Name()))
										result.FilesNotImported = append(result.FilesNotImported, osPathname)
									}
								}
							}
						}
					}
					return nil
				},
				Unsorted: true,
			})

			if err != nil {
				result.Errors = append(result.Errors, err)
			}

		}

	}
	return result
}

func importFromGoProV2(root string, output string, sortoptions SortOptions, cameraName string) utils.Result {
	mediaFolder := `\d\d\dGOPRO`

	fileTypes := []FileTypeMatch{
		{
			Regex:    regexp.MustCompile(`GOPR\d+.JPG`),
			Type:     Photo,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`GP\d+.JPG`),
			Type:     Photo,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`GX\d+.MP4`),
			Type:     Video,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`GH\d+.MP4`),
			Type:     Video,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`GL\d+.LRV`),
			Type:     LowResolutionVideo,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`GH\d+.THM`),
			Type:     Thumbnail,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`GX\d+.THM`),
			Type:     Thumbnail,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`GG\d+.MP4`), // Live Bursts...
			Type:     Video,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`G\d+.JPG`),
			Type:     Multishot,
			HeroMode: true,
		},
	}
	var result utils.Result
	/*
		The idea is to have a result like:

		20-02-2021/HERO9_Black/photos/
							GOPR00001.JPG
					   /videos
							single/
								GH010001.MP4
							proxy/
								GL010001.LRV
							thumbnails/
								GL010001.THM


	*/
	folders, err := ioutil.ReadDir(root)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return result
	}

	for _, f := range folders {
		r, err := regexp.MatchString(mediaFolder, f.Name())
		if err != nil {
			result.Errors = append(result.Errors, err)
		}
		if r {
			color.Green("Looking at %s", f.Name())

			err = godirwalk.Walk(filepath.Join(root, f.Name()), &godirwalk.Options{
				Callback: func(osPathname string, de *godirwalk.Dirent) error {

					for _, ftype := range fileTypes {
						if ftype.Regex.MatchString(de.Name()) {

							if sortoptions.ByDays {
								t, err := times.Stat(osPathname)
								if err != nil {
									log.Fatal(err.Error())
								}
								if t.HasBirthTime() {
									d := t.BirthTime()

									mediaDate := d.Format("02-01-2006")
									if strings.Contains(sortoptions.DateFormat, "year") && strings.Contains(sortoptions.DateFormat, "month") && strings.Contains(sortoptions.DateFormat, "day") {
										replacer := strings.NewReplacer("day", "02", "month", "01", "year", "2006")
										mediaDate = d.Format(replacer.Replace(sortoptions.DateFormat))
									}

									dayFolder := filepath.Join(output, mediaDate)
									if _, err := os.Stat(dayFolder); os.IsNotExist(err) {
										os.Mkdir(dayFolder, 0755)
									}

									if sortoptions.ByCamera {
										if _, err := os.Stat(filepath.Join(dayFolder, cameraName)); os.IsNotExist(err) {
											os.Mkdir(filepath.Join(dayFolder, cameraName), 0755)
										}
										dayFolder = filepath.Join(dayFolder, cameraName)
									}

									switch ftype.Type {
									case Video:
										x := de.Name()

										filename := fmt.Sprintf("%s%s-%s.%s", x[:2], x[4:][:4], x[2:][:2], strings.Split(x, ".")[1])
										color.Green(">>> %s", filename, color.Bold)

										if _, err := os.Stat(filepath.Join(dayFolder, "videos")); os.IsNotExist(err) {
											err = os.MkdirAll(filepath.Join(dayFolder, "videos"), 0755)
											if err != nil {
												log.Fatal(err.Error())
											}
										}

										err = utils.CopyFile(osPathname, filepath.Join(dayFolder, "videos", filename), 1000)
										if err != nil {
											result.Errors = append(result.Errors, err)
											result.FilesNotImported = append(result.FilesNotImported, osPathname)
										} else {
											result.FilesImported += 1
										}
									case Photo:
										if _, err := os.Stat(filepath.Join(dayFolder, "photos")); os.IsNotExist(err) {
											err = os.MkdirAll(filepath.Join(dayFolder, "photos"), 0755)
											if err != nil {
												log.Fatal(err.Error())
											}
										}

										color.Green(">>> %s", de.Name(), color.Bold)

										err = utils.CopyFile(osPathname, filepath.Join(dayFolder, "photos", de.Name()), 1000)
										if err != nil {
											result.Errors = append(result.Errors, err)
											result.FilesNotImported = append(result.FilesNotImported, osPathname)
										} else {
											result.FilesImported += 1
										}

									case LowResolutionVideo:
										if !sortoptions.SkipAuxiliaryFiles {
											if _, err := os.Stat(filepath.Join(dayFolder, "videos/proxy")); os.IsNotExist(err) {
												err = os.MkdirAll(filepath.Join(dayFolder, "videos/proxy"), 0755)
												if err != nil {
													log.Fatal(err.Error())
												}
											}

											x := de.Name()

											filename := fmt.Sprintf("%s%s-%s.%s", x[:2], x[4:][:4], x[2:][:2], strings.Split(x, ".")[1])
											err = utils.CopyFile(osPathname, filepath.Join(dayFolder, "videos/proxy", filename), 1000)
											if err != nil {
												result.Errors = append(result.Errors, err)
												result.FilesNotImported = append(result.FilesNotImported, osPathname)
											} else {
												result.FilesImported += 1
											}
										}
									case Thumbnail:
										if !sortoptions.SkipAuxiliaryFiles {
											if _, err := os.Stat(filepath.Join(dayFolder, "videos/proxy")); os.IsNotExist(err) {
												err = os.MkdirAll(filepath.Join(dayFolder, "videos/proxy"), 0755)
												if err != nil {
													log.Fatal(err.Error())
												}
											}

											x := de.Name()

											filename := fmt.Sprintf("%s%s-%s.%s", x[:2], x[4:][:4], x[2:][:2], strings.Split(x, ".")[1])
											err = utils.CopyFile(osPathname, filepath.Join(dayFolder, "videos/proxy", filename), 1000)
											if err != nil {
												result.Errors = append(result.Errors, err)
												result.FilesNotImported = append(result.FilesNotImported, osPathname)
											} else {
												result.FilesImported += 1
											}
										}
									case Multishot:
										root = de.Name()[:4]
										if _, err := os.Stat(filepath.Join(dayFolder, "multishot", root)); os.IsNotExist(err) {
											err = os.MkdirAll(filepath.Join(dayFolder, "multishot", root), 0755)
											if err != nil {
												log.Fatal(err.Error())
											}
										}

										color.Green(">>> %s/%s", root, de.Name(), color.Bold)

										err = utils.CopyFile(osPathname, filepath.Join(dayFolder, "multishot", root, de.Name()), 1000)
										if err != nil {
											result.Errors = append(result.Errors, err)
											result.FilesNotImported = append(result.FilesNotImported, osPathname)
										} else {
											result.FilesImported += 1
										}
									default:
										color.Red("Unsupported file %s", de.Name())
										result.Errors = append(result.Errors, errors.New("Unsupported file "+de.Name()))
										result.FilesNotImported = append(result.FilesNotImported, osPathname)
									}
								}
							}
						}
					}
					return nil
				},
				Unsorted: true,
			})

			if err != nil {
				result.Errors = append(result.Errors, err)
			}

		}

	}
	return result
}

func importFromGoProV1(root string, output string, sortoptions SortOptions, cameraName string) utils.Result {
	mediaFolder := `\d\d\dGOPRO`

	fileTypes := []FileTypeMatch{
		{
			Regex:    regexp.MustCompile(`GOPR\d+.JPG`),
			Type:     Photo,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`G\d+.JPG`),
			Type:     Multishot,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`GOPR\d+.MP4`),
			Type:     Video,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`GP\d+.MP4`),
			Type:     Video,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`GOPR\d+.LRV`),
			Type:     LowResolutionVideo,
			HeroMode: true,
		},
		{
			Regex:    regexp.MustCompile(`GOPR\d+.THM`),
			Type:     Thumbnail,
			HeroMode: true,
		},
	}
	var result utils.Result

	folders, err := ioutil.ReadDir(root)
	if err != nil {
		result.Errors = append(result.Errors, err)
		return result
	}

	for _, f := range folders {
		r, err := regexp.MatchString(mediaFolder, f.Name())
		if err != nil {
			result.Errors = append(result.Errors, err)
		}
		if r {
			color.Green("Looking at %s", f.Name())

			err = godirwalk.Walk(filepath.Join(root, f.Name()), &godirwalk.Options{
				Callback: func(osPathname string, de *godirwalk.Dirent) error {

					for _, ftype := range fileTypes {
						if ftype.Regex.MatchString(de.Name()) {

							if sortoptions.ByDays {
								t, err := times.Stat(osPathname)
								if err != nil {
									log.Fatal(err.Error())
								}
								if t.HasBirthTime() {
									d := t.BirthTime()

									mediaDate := d.Format("02-01-2006")
									if strings.Contains(sortoptions.DateFormat, "year") && strings.Contains(sortoptions.DateFormat, "month") && strings.Contains(sortoptions.DateFormat, "day") {
										replacer := strings.NewReplacer("day", "02", "month", "01", "year", "2006")
										mediaDate = d.Format(replacer.Replace(sortoptions.DateFormat))
									}

									dayFolder := filepath.Join(output, mediaDate)
									if _, err := os.Stat(dayFolder); os.IsNotExist(err) {
										os.Mkdir(dayFolder, 0755)
									}

									if sortoptions.ByCamera {
										if _, err := os.Stat(filepath.Join(dayFolder, cameraName)); os.IsNotExist(err) {
											os.Mkdir(filepath.Join(dayFolder, cameraName), 0755)
										}
										dayFolder = filepath.Join(dayFolder, cameraName)
									}

									switch ftype.Type {
									case Video:
										x := de.Name()

										chaptered := regexp.MustCompile(`GP\d+.MP4`)
										if chaptered.MatchString(de.Name()) {
											x = fmt.Sprintf("GOPR%s%s.%s", x[4:][:4], x[2:][:2], strings.Split(x, ".")[1])
										}
										color.Green(">>> %s", x, color.Bold)

										if _, err := os.Stat(filepath.Join(dayFolder, "videos")); os.IsNotExist(err) {
											err = os.MkdirAll(filepath.Join(dayFolder, "videos"), 0755)
											if err != nil {
												log.Fatal(err.Error())
											}
										}

										err = utils.CopyFile(osPathname, filepath.Join(dayFolder, "videos", x), 1000)
										if err != nil {
											result.Errors = append(result.Errors, err)
											result.FilesNotImported = append(result.FilesNotImported, osPathname)
										} else {
											result.FilesImported += 1
										}
									case Photo:
										if _, err := os.Stat(filepath.Join(dayFolder, "photos")); os.IsNotExist(err) {
											err = os.MkdirAll(filepath.Join(dayFolder, "photos"), 0755)
											if err != nil {
												log.Fatal(err.Error())
											}
										}

										color.Green(">>> %s", de.Name(), color.Bold)

										err = utils.CopyFile(osPathname, filepath.Join(dayFolder, "photos", de.Name()), 1000)
										if err != nil {
											result.Errors = append(result.Errors, err)
											result.FilesNotImported = append(result.FilesNotImported, osPathname)
										} else {
											result.FilesImported += 1
										}

									case LowResolutionVideo:
										if !sortoptions.SkipAuxiliaryFiles {
											if _, err := os.Stat(filepath.Join(dayFolder, "videos/proxy")); os.IsNotExist(err) {
												err = os.MkdirAll(filepath.Join(dayFolder, "videos/proxy"), 0755)
												if err != nil {
													log.Fatal(err.Error())
												}
											}

											x := de.Name()

											err = utils.CopyFile(osPathname, filepath.Join(dayFolder, "videos/proxy", x), 1000)
											if err != nil {
												result.Errors = append(result.Errors, err)
												result.FilesNotImported = append(result.FilesNotImported, osPathname)
											} else {
												result.FilesImported += 1
											}
										}
									case Thumbnail:
										if !sortoptions.SkipAuxiliaryFiles {
											if _, err := os.Stat(filepath.Join(dayFolder, "videos/proxy")); os.IsNotExist(err) {
												err = os.MkdirAll(filepath.Join(dayFolder, "videos/proxy"), 0755)
												if err != nil {
													log.Fatal(err.Error())
												}
											}

											x := de.Name()
											err = utils.CopyFile(osPathname, filepath.Join(dayFolder, "videos/proxy", x), 1000)
											if err != nil {
												result.Errors = append(result.Errors, err)
												result.FilesNotImported = append(result.FilesNotImported, osPathname)
											} else {
												result.FilesImported += 1
											}
										}
									case Multishot:
										root = de.Name()[:4]
										if _, err := os.Stat(filepath.Join(dayFolder, "multishot", root)); os.IsNotExist(err) {
											err = os.MkdirAll(filepath.Join(dayFolder, "multishot", root), 0755)
											if err != nil {
												log.Fatal(err.Error())
											}
										}

										color.Green(">>> %s/%s", root, de.Name(), color.Bold)

										err = utils.CopyFile(osPathname, filepath.Join(dayFolder, "multishot", root, de.Name()), 1000)
										if err != nil {
											result.Errors = append(result.Errors, err)
											result.FilesNotImported = append(result.FilesNotImported, osPathname)
										} else {
											result.FilesImported += 1
										}
									default:
										color.Red("Unsupported file %s", de.Name())
										result.Errors = append(result.Errors, errors.New("Unsupported file "+de.Name()))
										result.FilesNotImported = append(result.FilesNotImported, osPathname)
									}
								}
							}
						}
					}
					return nil
				},
				Unsorted: true,
			})

			if err != nil {
				result.Errors = append(result.Errors, err)
			}

		}

	}
	return result
}

/*
GoPro adds a trailing comma to their version.txt file... this removes it.
*/
func cleanVersion(s string) string {
	i := strings.LastIndex(s, ",")
	excludingLast := s[:i] + strings.Replace(s[i:], ",", "", 1)
	return excludingLast
}

func readInfo(in string) (*GoProVersion, error) {
	files, err := ioutil.ReadDir(in)
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		if f.Name() == fmt.Sprint(MISC) {

			filesInMisc, err := ioutil.ReadDir(in + "/MISC")
			if err != nil {
				return nil, err
			}
			for _, f := range filesInMisc {
				if f.Name() == fmt.Sprint(Version) {
					jsonFile, err := os.Open(in + "/MISC/" + fmt.Sprint(Version))
					if err != nil {
						return nil, err
					}
					inBytes, err := ioutil.ReadAll(jsonFile)
					if err != nil {
						return nil, err
					}
					text := string(inBytes)
					clean := cleanVersion(text)
					var gpVersion GoProVersion
					err = json.Unmarshal([]byte(clean), &gpVersion)
					if err != nil {
						return nil, err
					}
					return &gpVersion, nil

				}
			}
		}
	}
	return nil, errors.New("MISC not found")
}
