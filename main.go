package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io/fs"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	wr "github.com/mroth/weightedrand"
	"github.com/urfave/cli/v2"
)

// Base NFT Structure
type Nft struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Attributes  []Attribute `json:"attributes"`
	Image       string      `json:"image"`
	Description string      `json:"description"`
}

type Attribute struct {
	TraitType string `json:"trait_type"`
	Value     string `json:"value"`
}

type store struct {
	Background Background
}

type Background struct {
	Linework []string
	Fill     []string
}

type Gender int

const (
	MALE Gender = iota
	FEMALE
)

//go:embed data/*
var dataFS embed.FS

var assetsStore *store

type Metadata struct {
	Linework string
	Fill     string
}

// Ape Attributes as Struct
type nft struct {
	Lineworks []string
	FIlls     []string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func init() {
	background := getBackground()
	assetsStore = &store{
		Background: background, // Background Traits + Can be be unisex traits + Add assets to store as necessary
	}

	rand.Seed(time.Now().UTC().UnixNano()) // Sample randomneess seed
}

func main() {
	app := cli.NewApp()
	app.Name = "nftgen"
	app.Usage = "NFT generator service CLI"
	app.Commands = []*cli.Command{
		{
			Name:    "generate",
			Aliases: []string{"g"},
			Usage:   "Generates random nft",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "output,o",
					Value: "nft.png",
					Usage: "Output file name",
				},
			},
			Action: func(c *cli.Context) error {
				return GenerateFile(c.String("output"))
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func generateRandomString() string {
	rand.Seed(time.Now().Unix())
	var output strings.Builder

	charSet := []rune("abcdedfghijklmnopqrstABCDEFGHIJKLMNOP£")
	length := 20
	for i := 0; i < length; i++ {
		random := rand.Intn(len(charSet))
		randomChar := charSet[random]
		output.WriteRune(randomChar)
	}

	return output.String()
}

func Generate() (image.Image, Metadata, error) {
	rand.Seed(time.Now().UnixNano())
	h := fnv.New32a()

	_, err := h.Write([]byte(string(generateRandomString())))

	check(err)
	return randomNFT(nft{}, int64(h.Sum32()))
}

func GenerateFile(filePath string) error {
	img, metadata, err := Generate()
	check(err)

	return saveToFile(img, filePath, metadata)
}

func randomNFT(n nft, seed int64) (img image.Image, meta Metadata, err error) {

	nftImage := image.NewRGBA(image.Rect(0, 0, 4500, 1500))

	// Generate Background
	lineworkChoice := selectLinework(assetsStore.Background.Linework)
	fmt.Println("lineworkChoice: " + lineworkChoice)
	lineworkChoicePath := ""
	for i := 0; i < len(assetsStore.Background.Linework); i++ {
		if formatProperty(trimProperty(assetsStore.Background.Linework[i])) == lineworkChoice {
			lineworkChoicePath = assetsStore.Background.Linework[i]
		}
	}
	linework := lineworkChoicePath

	fillChoice := selectFill(assetsStore.Background.Fill)
	fmt.Println("fillChoice: " + fillChoice)
	fillChoicePath := ""
	for i := 0; i < len(assetsStore.Background.Fill); i++ {
		if formatProperty(trimProperty(assetsStore.Background.Fill[i])) == fillChoice {
			fillChoicePath = assetsStore.Background.Fill[i]
		}
	}
	fill := fillChoicePath

	// Generate Metadata
	meta = Metadata{
		Linework: formatProperty(trimProperty(linework)),
		Fill:     formatProperty(trimProperty(fill)),
	}

	// Can use conditionals with string matching to affect draw order here
	err = drawImage(nftImage, fill, err)
	err = drawImage(nftImage, linework, err)

	return nftImage, meta, err
}

func getNFT() nft {
	assetList := nft{}
	return assetList
}

type Rarity int

// Rarity Percentages
const (
	LEGENDARY = 1
	EPIC      = 2
	RARE      = 3
	UNCOMMON  = 5
	COMMON    = 10
)

func selectFill(fills []string) string {
	rand.Seed(time.Now().UTC().UnixNano())

	c, err := wr.NewChooser(
		wr.Choice{Item: "Abyssal", Weight: COMMON},
		wr.Choice{Item: "Amnesia", Weight: COMMON},
		wr.Choice{Item: "Azure", Weight: COMMON},
		wr.Choice{Item: "Coral", Weight: COMMON},
		wr.Choice{Item: "Emerald", Weight: COMMON},
		wr.Choice{Item: "Graphite", Weight: COMMON},
		wr.Choice{Item: "Inferno", Weight: COMMON},
		wr.Choice{Item: "Mystic", Weight: COMMON},
		wr.Choice{Item: "Mythic", Weight: COMMON},
		wr.Choice{Item: "Neon", Weight: COMMON},
		wr.Choice{Item: "Peach", Weight: COMMON},
		wr.Choice{Item: "Sea Storm", Weight: COMMON},
		wr.Choice{Item: "Slime", Weight: COMMON},
		wr.Choice{Item: "Sunshine", Weight: COMMON},
		wr.Choice{Item: "Wildfire", Weight: COMMON},
	)
	check(err)

	return c.Pick().(string)
}

func selectLinework(lineworks []string) string {
	rand.Seed(time.Now().UTC().UnixNano())

	c, err := wr.NewChooser(
		wr.Choice{Item: "Linework", Weight: COMMON},
	)
	check(err)

	return c.Pick().(string)
}

func getBackground() Background {
	bg := Background{
		Linework: importAssetNames("data/banners"),
		Fill:     importAssetNames("data/banners"),
	}
	return bg
}

func saveToFile(img image.Image, filePath string, meta Metadata) error {
	outFile, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".jpeg", ".jpg":
		err = jpeg.Encode(outFile, img, &jpeg.Options{Quality: 95})
		check(err)
	case ".gif":
		err = gif.Encode(outFile, img, nil)
		check(err)
	default:
		err = png.Encode(outFile, img)
		check(err)
	}

	attributes := make([]Attribute, 0)

	linework := Attribute{
		TraitType: "Linework",
		Value:     meta.Linework,
	}

	fill := Attribute{
		TraitType: "Fill",
		Value:     meta.Fill,
	}

	attributes = append(attributes, linework)
	attributes = append(attributes, fill)

	// File Count Directory
	count, err := getFileCount()
	if err != nil {
		fmt.Println("Could not find file count")
	}

	ipfsSampleMeta := Nft{
		ID:          fmt.Sprintf("%d", count),
		Name:        fmt.Sprintf("Banner Example #%d", count),
		Attributes:  attributes,
		Image:       "{IPFS_IMAGE_URL}", // String replace w/ script after CID is generated
		Description: "Description",
	}

	metaPath := trimExtension(filePath)
	metaPath = fmt.Sprintf("%s.json", metaPath)
	metaFile, err := json.MarshalIndent(ipfsSampleMeta, "", "")
	if err != nil {
		fmt.Println(err)
	}

	_ = ioutil.WriteFile(metaPath, metaFile, 0777)

	return err
}

func importAssetNames(dir string) []string {
	dirEntries, err := fs.ReadDir(dataFS, dir)
	if err != nil {
		panic(err)
	}
	assets := make([]string, len(dirEntries))
	for i, dirEntry := range dirEntries {
		assets[i] = path.Join(dir, dirEntry.Name())
	}
	sort.Sort(naturalSort(assets))
	return assets
}

// Imports an asset from a file
func importAsset(fPath string) []byte {
	b, err := fs.ReadFile(dataFS, fPath)
	check(err)
	return b
}

type naturalSort []string

var r = regexp.MustCompile(`[^0-9]+|[0-9]+`)

func (s naturalSort) Len() int {
	return len(s)
}

func (s naturalSort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s naturalSort) Less(i, j int) bool {
	spliti := r.FindAllString(strings.Replace(s[i], " ", "", -1), -1)
	splitj := r.FindAllString(strings.Replace(s[j], " ", "", -1), -1)

	for index := 0; index < len(spliti) && index < len(splitj); index++ {
		if spliti[index] != splitj[index] {
			// Both slices are numbers
			if isNumber(spliti[index][0]) && isNumber(splitj[index][0]) {
				// Remove Leading Zeroes
				stringi := strings.TrimLeft(spliti[index], "0")
				stringj := strings.TrimLeft(splitj[index], "0")
				if len(stringi) == len(stringj) {
					for indexchar := 0; indexchar < len(stringi); indexchar++ {
						if stringi[indexchar] != stringj[indexchar] {
							return stringi[indexchar] < stringj[indexchar]
						}
					}
					return len(spliti[index]) < len(splitj[index])
				}
				return len(stringi) < len(stringj)
			}

			if isNumber(spliti[index][0]) || isNumber(splitj[index][0]) {
				return isNumber(spliti[index][0])
			}

			return spliti[index] < splitj[index]
		}

	}

	for index := 0; index < len(s[i]) && index < len(s[j]); index++ {
		if isNumber(s[i][index]) || isNumber(s[j][index]) {
			return isNumber(s[i][index])
		}
	}
	return s[i] < s[j]
}

func isNumber(input uint8) bool {
	return input >= '0' && input <= '9'
}

func drawImage(dst draw.Image, asset string, err error) error {
	check(err)

	src, _, err := image.Decode(bytes.NewReader(importAsset(asset)))
	check(err)

	draw.Draw(dst, dst.Bounds(), src, image.Point{0, 0}, draw.Over)
	return nil
}

func formatProperty(trait string) string {
	replacedString := trait
	if trait != "2D" {
		replacedString = strings.Replace(trait, "-", " ", -1)
		replacedString = strings.ToLower(replacedString)
		replacedString = strings.Title(replacedString)
	}
	return replacedString
}

func trimProperty(property string) string {
	trimmed1 := trimFilePath(property)
	trimmed := trimExtension(trimmed1)
	return trimmed
}

func trimExtension(filename string) (trimmed string) {
	name := strings.TrimSuffix(filename, filepath.Ext(filename))
	return name
}

func trimFilePath(path string) (filename string) {
	file := filepath.Base(path)
	return file
}

func getFileCount() (count int, err error) {
	path := "./output"
	i := 1
	files, err := ioutil.ReadDir(path)
	if err != nil {
		fmt.Println(err)
		return 1, err
	}
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".png" {
			i++
		}
	}
	return i, nil
}
