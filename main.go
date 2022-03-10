package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
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

var ErrUnsupportedGender = errors.New("unsupported gender")

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
	Male       nft
	Female     nft
}

type Background struct {
	CompositeTraitOne   []string
	CompositeTraitTwo   []string
	CompositeTraitThree []string
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
	Gender              string
	CompositeTraitOne   string
	CompositeTraitTwo   string
	CompositeTraitThree string
	TraitOne            string
	TraitOneDependent   string
	TraitThree          string
	// Gender         string
	// Canvas         string
	// BackgroundType string
	// BackgroundFill string
	// FurType        string
	// FurFill        string
	// Eyes           string
	// Accessories    string
	// Hair           string
	// Mouth          string
	// Earrings       string
	// Hats           string
}

// Ape Attributes as Struct
type nft struct {
	AllTraitOneOptions          []string
	AllTraitOneDependentOptions []string
	AllTraitThreeOptions        []string
	// FurType     []string
	// FurFill     []string
	// Eyes        []string
	// Hair        []string
	// Mouth       []string
	// Earrings    []string
	// Hats        []string
	// Accessories []string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func init() {
	male := getNFT(MALE)
	female := getNFT(FEMALE)
	background := getBackground()
	assetsStore = &store{
		Male:       male,       // Male Specific Traits
		Female:     female,     // Female Specific Traits
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
			Name:      "generate",
			ArgsUsage: "<(male|m)|(female|f)>",
			Aliases:   []string{"g"},
			Usage:     "Generates random avatar",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "output,o",
					Value: "nft.png",
					Usage: "Output file name",
				},
			},
			Action: func(c *cli.Context) error {
				var g Gender
				var err error
				switch c.Args().First() {
				case "male", "m":
					g = MALE
				case "female", "f":
					g = FEMALE
				default:
					return fmt.Errorf("incorrect gender param. Run `nftgen help generate`")
				}

				username := c.String("username")
				if username != "" {
					err = GenerateFile(g, c.String("output"))
				} else {
					err = GenerateFile(g, c.String("output"))
				}
				return err
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

func Generate(gender Gender) (image.Image, Metadata, error) {
	rand.Seed(time.Now().UnixNano())
	h := fnv.New32a()

	_, err := h.Write([]byte(string(generateRandomString())))

	check(err)
	switch gender {
	case MALE:
		return randomNFT(assetsStore.Male, int64(h.Sum32()), MALE)
	case FEMALE:
		return randomNFT(assetsStore.Female, int64(h.Sum32()), FEMALE)
	default:
		return nil, Metadata{}, ErrUnsupportedGender
	}
}

func GenerateFile(gender Gender, filePath string) error {
	img, metadata, err := Generate(gender)
	check(err)

	return saveToFile(img, filePath, metadata)
}

func randomNFT(n nft, seed int64, gender Gender) (img image.Image, meta Metadata, err error) {
	var genderPath string
	switch gender {
	case FEMALE:
		genderPath = "female"
	case MALE:
		genderPath = "male"
	}

	nftImage := image.NewRGBA(image.Rect(0, 0, 4000, 4000))

	// Generate Background
	compositeTraitOneChoice := selectTraitOne(assetsStore.Background.CompositeTraitOne)
	fmt.Println("compositeTraitOneChoice: " + compositeTraitOneChoice)
	compositeTraitOneChoicePath := ""
	for i := 0; i < len(assetsStore.Background.CompositeTraitOne); i++ {
		if formatProperty(trimProperty(assetsStore.Background.CompositeTraitOne[i])) == compositeTraitOneChoice {
			compositeTraitOneChoicePath = assetsStore.Background.CompositeTraitOne[i]
		}
	}
	compositeTraitOne := compositeTraitOneChoicePath

	compositeTraitTwoChoice := selectCompositeTraitTwo(assetsStore.Background.CompositeTraitTwo)
	fmt.Println("compositeTraitTwoChoice: " + compositeTraitTwoChoice)
	compositeTraitTwoChoicePath := ""
	for i := 0; i < len(assetsStore.Background.CompositeTraitTwo); i++ {
		if formatProperty(trimProperty(assetsStore.Background.CompositeTraitTwo[i])) == compositeTraitTwoChoice {
			compositeTraitTwoChoicePath = assetsStore.Background.CompositeTraitTwo[i]
		}
	}
	compositeTraitTwo := compositeTraitTwoChoicePath

	assetsStore.Background.CompositeTraitThree = importAssetNames("data/backgrounds/composite-trait-two/" + trimProperty(compositeTraitTwo))
	CompositeTraitThreeChoice := selectCompositeTraitThree(compositeTraitTwoChoice, assetsStore.Background.CompositeTraitThree)
	fmt.Println("CompositeTraitThreeChoice: " + CompositeTraitThreeChoice)
	CompositeTraitThreeChoicePath := ""
	for i := 0; i < len(assetsStore.Background.CompositeTraitThree); i++ {
		if formatProperty(trimProperty(assetsStore.Background.CompositeTraitThree[i])) == CompositeTraitThreeChoice {
			CompositeTraitThreeChoicePath = assetsStore.Background.CompositeTraitThree[i]
		}
	}
	compositeTraitThree := CompositeTraitThreeChoicePath

	// Generate Base Traits
	traitOneChoice := selectTraitOne(n.AllTraitOneOptions)
	fmt.Println("TraitOne: " + traitOneChoice)
	traitOneChoicePath := ""
	for i := 0; i < len(n.AllTraitOneOptions); i++ {
		if formatProperty(trimProperty(n.AllTraitOneOptions[i])) == traitOneChoice {
			traitOneChoicePath = n.AllTraitOneOptions[i]
		}
	}
	traitOne := traitOneChoicePath

	// Generate Trait Two Dependent on Trait One
	n.AllTraitOneDependentOptions = importAssetNames("data/" + genderPath + "/trait-one-dependent/" + trimProperty(strings.ToLower(fmt.Sprintf("%s", traitOne))))
	traitOneDependentTraitChoice := selectTraitOneDependentTrait(traitOneChoice, n.AllTraitOneDependentOptions)
	fmt.Println("TraitOneDendentTrait Options: " + traitOneDependentTraitChoice)
	traitOneDependentTraitChoicePath := ""
	for i := 0; i < len(n.AllTraitOneDependentOptions); i++ {
		if formatProperty(trimProperty(n.AllTraitOneDependentOptions[i])) == traitOneDependentTraitChoice {
			traitOneDependentTraitChoicePath = n.AllTraitOneDependentOptions[i]
		}
	}
	traitOneDependentTrait := traitOneDependentTraitChoicePath

	traitThreeChoice := selectTraitThree(n.AllTraitThreeOptions)
	fmt.Println("Eyes: " + traitThreeChoice)
	traitThreeChoicePath := ""
	for i := 0; i < len(n.AllTraitThreeOptions); i++ {
		if formatProperty(trimProperty(n.AllTraitThreeOptions[i])) == traitThreeChoice {
			traitThreeChoicePath = n.AllTraitThreeOptions[i]
		}
	}
	traitThree := traitThreeChoicePath

	// Generate Metadata
	meta = Metadata{
		Gender:              strings.Title(genderPath),
		CompositeTraitOne:   formatProperty(trimProperty(compositeTraitOne)),
		CompositeTraitTwo:   formatProperty(trimProperty(compositeTraitTwo)),
		CompositeTraitThree: formatProperty(trimProperty(compositeTraitThree)),
		TraitOne:            formatProperty(trimProperty(traitOne)),
		TraitOneDependent:   formatProperty(trimProperty(traitOneDependentTrait)),
		TraitThree:          formatProperty(trimProperty(traitThree)),
	}

	// Can use conditionals with string matching to affect draw order here
	err = drawImage(nftImage, compositeTraitOne, err)
	err = drawImage(nftImage, compositeTraitTwo, err)
	err = drawImage(nftImage, compositeTraitThree, err)
	err = drawImage(nftImage, traitOne, err)
	err = drawImage(nftImage, traitOneDependentTrait, err)
	err = drawImage(nftImage, traitThree, err)

	return nftImage, meta, err
}

func getNFT(gender Gender) nft {
	var genderPath string

	switch gender {
	case FEMALE:
		genderPath = "female"
	case MALE:
		genderPath = "male"
	}

	assetList := nft{
		AllTraitOneOptions:          importAssetNames("data/" + genderPath + "/trait-one-options"),
		AllTraitOneDependentOptions: importAssetNames("data/" + genderPath + "/trait-one-dependent-options"),
		AllTraitThreeOptions:        importAssetNames("data/" + genderPath + "/trait-three-options"),
	}
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

func selectcompositeTraitOne(canvases []string) string {
	rand.Seed(time.Now().UTC().UnixNano())

	c, err := wr.NewChooser(
		wr.Choice{Item: "Option 1", Weight: EPIC},
		wr.Choice{Item: "Option 2", Weight: COMMON},
		wr.Choice{Item: "Option 3", Weight: RARE},
		wr.Choice{Item: "Option 4", Weight: COMMON},
		wr.Choice{Item: "Option 5", Weight: RARE},
		wr.Choice{Item: "Option 6", Weight: COMMON},
		wr.Choice{Item: "Option 7", Weight: COMMON},
	)
	check(err)

	return c.Pick().(string)
}

func selectCompositeTraitTwo(compositeTraitTwos []string) string {
	rand.Seed(time.Now().UTC().UnixNano())

	c, err := wr.NewChooser(
		wr.Choice{Item: "Option 1", Weight: COMMON},
		wr.Choice{Item: "Option 2", Weight: COMMON},
		wr.Choice{Item: "Option 3", Weight: COMMON},
	)
	check(err)

	return c.Pick().(string)
}

func selectCompositeTraitThree(compositeTraitTwo string, backgroundFills []string) string {
	rand.Seed(time.Now().UTC().UnixNano())

	switch {
	// Nested rarity Example
	case compositeTraitTwo == "Option 1":
		c, err := wr.NewChooser(
			wr.Choice{Item: "Option 1a", Weight: COMMON},
			wr.Choice{Item: "Option 2a", Weight: COMMON},
			wr.Choice{Item: "Option 3a", Weight: COMMON},
		)
		check(err)
		return c.Pick().(string)
	case compositeTraitTwo == "Option 2":
		c, err := wr.NewChooser(
			wr.Choice{Item: "Option 2a", Weight: COMMON},
			wr.Choice{Item: "Option 2b", Weight: COMMON},
			wr.Choice{Item: "Option 2c", Weight: COMMON},
		)
		check(err)
		return c.Pick().(string)
	case compositeTraitTwo == "Option 3":
		c, err := wr.NewChooser(
			wr.Choice{Item: "Option 3a", Weight: COMMON},
			wr.Choice{Item: "Option 3b", Weight: COMMON},
			wr.Choice{Item: "Option 3c", Weight: COMMON},
		)
		check(err)
		return c.Pick().(string)
	}

	return ""
}

func selectTraitOne(traitOnes []string) string {
	rand.Seed(time.Now().UTC().UnixNano())

	c, err := wr.NewChooser(
		wr.Choice{Item: "Option 1", Weight: RARE},
		wr.Choice{Item: "Option 2", Weight: UNCOMMON},
		wr.Choice{Item: "Option 3", Weight: EPIC},
		wr.Choice{Item: "Option 4", Weight: COMMON},
		wr.Choice{Item: "Option 5", Weight: UNCOMMON},
		wr.Choice{Item: "Option 6", Weight: COMMON},
		wr.Choice{Item: "Option 7", Weight: UNCOMMON},
	)
	check(err)

	return c.Pick().(string)
}

func selectTraitOneDependentTrait(compositeTraitOne string, AllCompositeTraitOneDependents []string) string {
	rand.Seed(time.Now().UTC().UnixNano())

	switch {
	// Nested rarity Example
	case compositeTraitOne == "Option 1":
		c, err := wr.NewChooser(
			wr.Choice{Item: "Option 1a", Weight: COMMON},
			wr.Choice{Item: "Option 2a", Weight: COMMON},
			wr.Choice{Item: "Option 3a", Weight: COMMON},
		)
		check(err)
		return c.Pick().(string)
	case compositeTraitOne == "Option 2":
		c, err := wr.NewChooser(
			wr.Choice{Item: "Option 2a", Weight: COMMON},
			wr.Choice{Item: "Option 2b", Weight: COMMON},
			wr.Choice{Item: "Option 2c", Weight: COMMON},
		)
		check(err)
		return c.Pick().(string)
	case compositeTraitOne == "Option 3":
		c, err := wr.NewChooser(
			wr.Choice{Item: "Option 3a", Weight: COMMON},
			wr.Choice{Item: "Option 3b", Weight: COMMON},
			wr.Choice{Item: "Option 3c", Weight: COMMON},
		)
		check(err)
		return c.Pick().(string)
	}

	return ""
}

func selectTraitThree(traitThrees []string) string {
	rand.Seed(time.Now().UTC().UnixNano())

	c, err := wr.NewChooser(
		wr.Choice{Item: "Trait Three Option 1", Weight: COMMON},
		wr.Choice{Item: "Trait Three Option 2", Weight: UNCOMMON},
		wr.Choice{Item: "Trait Three Option 3", Weight: UNCOMMON},
		wr.Choice{Item: "Trait Three Option 4", Weight: RARE},
		wr.Choice{Item: "Trait Three Option 5", Weight: COMMON},
		wr.Choice{Item: "Trait Three Option 6", Weight: COMMON},
		wr.Choice{Item: "Trait Three Option 7", Weight: LEGENDARY},
		wr.Choice{Item: "Trait Three Option 8", Weight: LEGENDARY},
		wr.Choice{Item: "Trait Three Option 9", Weight: EPIC},
	)
	check(err)

	return c.Pick().(string)
}

func getBackground() Background {
	bg := Background{
		CompositeTraitOne: importAssetNames("data/backgrounds/composite-trait-one"),
		CompositeTraitTwo: importAssetNames("data/backgrounds/composite-trait-two"),
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

	gender := Attribute{
		TraitType: "Gender",
		Value:     meta.Gender,
	}

	compositeTraitOne := Attribute{
		TraitType: "Composite Trait One",
		Value:     meta.CompositeTraitOne,
	}

	compositeTraitTwo := Attribute{
		TraitType: "Composite Trait Two",
		Value:     meta.CompositeTraitTwo,
	}

	compositeTraitThree := Attribute{
		TraitType: "Composite Trait Three",
		Value:     meta.CompositeTraitThree,
	}

	traitOne := Attribute{
		TraitType: "Trait One",
		Value:     meta.TraitOne,
	}

	TraitOneDependentTrait := Attribute{
		TraitType: "Trait Two",
		Value:     meta.TraitOneDependent,
	}

	traitThree := Attribute{
		TraitType: "Trait Three",
		Value:     meta.TraitThree,
	}

	attributes = append(attributes, gender)
	attributes = append(attributes, compositeTraitOne)
	attributes = append(attributes, compositeTraitTwo)
	attributes = append(attributes, compositeTraitThree)
	attributes = append(attributes, traitOne)
	attributes = append(attributes, TraitOneDependentTrait)
	attributes = append(attributes, traitThree)

	// File Count Directory
	count, err := getFileCount()
	if err != nil {
		fmt.Println("Could not find file count")
	}

	ipfsSampleMeta := Nft{
		ID:          fmt.Sprintf("%d", count),
		Name:        fmt.Sprintf("NFT Collection Title #%d", count),
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

func randInt(rnd *rand.Rand, min int, max int) int {
	return min + rnd.Intn(max-min)
}

// randStringSliceItem returns random element from slice of string
func randStringSliceItem(rnd *rand.Rand, slice []string) string {
	return slice[randInt(rnd, 0, len(slice))]
}

func jsonPrettyPrint(in string) string {
	var out bytes.Buffer
	err := json.Indent(&out, []byte(in), "", "\t")
	if err != nil {
		return in
	}
	return out.String()
}
