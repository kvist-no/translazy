package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/urfave/cli/v2"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

var token string

func main() {
	var sourceLanguage string
	var translationKey string
	var targetLanguages cli.StringSlice
	var syncTranslations bool

	app := &cli.App{
		Name:  "translazy",
		Usage: "Help you add new translations to your projects. It expects to find a DeepL API key in the file ~/.config/translazy/token.",
		Action: func(context *cli.Context) error {
			input := context.Args().First()

			translations := []Translation{
				{
					Lang: sourceLanguage,
					Text: input,
				},
			}

			for _, lang := range targetLanguages.Value() {
				translation, err := translate(input, sourceLanguage, lang)
				if err != nil {
					log.Fatal(err)
				}
				translations = append(translations, translation)
			}

			persistToLocaleFiles(translations, translationKey)

			if syncTranslations {
				syncPnpmLocales()
			}

			outputResults(translationKey, translations)

			return nil
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "source-lang",
				Aliases:     []string{"s"},
				Value:       "en",
				Usage:       "The language of the source text. This is a required parameter.",
				Destination: &sourceLanguage,
			},
			&cli.StringSliceFlag{
				Name:        "target-langs",
				Value:       cli.NewStringSlice("sv", "nb"),
				Usage:       "The languages to translate the source text to. This is a required parameter.",
				Destination: &targetLanguages,
			},
			&cli.StringFlag{
				Name:        "key",
				Aliases:     []string{"k"},
				Required:    true,
				Usage:       "The translation key to output translations to. This is a required parameter.",
				Destination: &translationKey,
			},
			&cli.BoolFlag{
				Name:        "sync",
				Usage:       "If set, the translations will be synced using the locale:import pnpm script.",
				Destination: &syncTranslations,
			},
			&cli.StringFlag{
				Name:        "token",
				Usage:       "The DeepL API key. If not set, the program will look for the key in the file ~/.config/translazy/token.",
				EnvVars:     []string{"DEEPL_API_KEY"},
				Destination: &token,
				FilePath:    os.Getenv("HOME") + "/.config/translazy/token",
				Required:    true,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func outputResults(translationKey string, translations []Translation) {
	fmt.Printf("Translations done for key \"%s\"\n", translationKey)
	for _, translation := range translations {
		fmt.Printf("  %s: %s\n", translation.Lang, translation.Text)
	}
}

func syncPnpmLocales() {
	err := exec.Command("pnpm", "run", "locale:import").Run()
	if err != nil {
		log.Fatal(err)
	}
}

type Translation struct {
	Lang string
	Text string `json:"text"`
}

type DeepLAPIInput struct {
	Text       []string `json:"text"`
	TargetLang string   `json:"target_lang"`
	SourceLang string   `json:"source_lang"`
}

type DeepLAPITranslation struct {
	DetectedSourceLanguage string `json:"detected_source_language"`
	Text                   string `json:"text"`
}

type DeepLAPIOutput struct {
	Translations []DeepLAPITranslation `json:"translations"`
}

func translate(text string, fromLang string, toLang string) (Translation, error) {
	postBody, _ := json.Marshal(DeepLAPIInput{
		Text:       []string{text},
		TargetLang: toLang,
		SourceLang: fromLang,
	})
	responseBody := bytes.NewBuffer(postBody)

	request, _ := http.NewRequest("POST", "https://api-free.deepl.com/v2/translate", responseBody)
	request.Header.Add("Authorization", "DeepL-Auth-Key "+strings.TrimSpace(token))
	request.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)

	if err != nil {
		return Translation{}, err
	}

	// parse response body to expected type
	var translation DeepLAPIOutput
	_ = json.NewDecoder(response.Body).Decode(&translation)

	if len(translation.Translations) == 0 {
		return Translation{}, fmt.Errorf("no translations found")
	}

	return Translation{
		Text: translation.Translations[0].Text,
		Lang: toLang,
	}, nil
}

func persistToLocaleFiles(translations []Translation, key string) {
	for _, translation := range translations {
		outputPath := "locale/" + norwegianConfusionHack(translation.Lang) + ".json"
		fileContent, _ := os.ReadFile(outputPath)
		var existingTranslations map[string]string
		_ = json.Unmarshal(fileContent, &existingTranslations)
		existingTranslations[key] = translation.Text
		updatedFileContent, _ := json.MarshalIndent(existingTranslations, "", "  ")
		_ = os.WriteFile(outputPath, updatedFileContent, 0644)
	}
}

// We've called the file "no" in our codebase, even though "nb" is more correct
func norwegianConfusionHack(lang string) string {
	if lang == "nb" {
		return "no"
	}

	return lang
}
