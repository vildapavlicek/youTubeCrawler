package parsers

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

// YoutubeParser has *logrus.Logger and data parsing method
type YoutubeParser struct {
	Log *logrus.Logger
}

// DataParser interface for data parsing
type DataParser interface {
	ParseData(response *http.Response) (link, title string, err error)
}

// ParseData parses youTube html for next video sufix and title
func (y YoutubeParser) ParseData(res *http.Response) (title, link string, err error) {
	defer res.Body.Close()
	doc, err := html.Parse(res.Body)
	if err != nil {
		y.Log.WithFields(logrus.Fields{
			"method": "html.Parse",
			"err":    err.Error(),
		}).Error("Failed to parse req.Body")
		return "", "", err
	}
	title, link, err = parseNode(doc)
	y.Log.WithFields(logrus.Fields{
		"method":      "ParseData",
		"parsedTitle": title,
		"parsedLink":  link,
	}).Trace("Parsed values at ParseData from parseNode(doc)")

	if err != nil {
		y.Log.WithFields(logrus.Fields{
			"method":      "ParseData",
			"parsedTitle": title,
			"parsedLink":  link,
			"err":         err.Error(),
		}).Debug("Error parsing title and link")
	}

	return title, link, err
}

func parseNode(n *html.Node) (title, link string, err error) {
	if n.Type == html.ElementNode && n.Data == "ul" {
		for _, v := range n.Attr {
			if v.Key == "class" && v.Val == "video-list" {
				title, link, err = parseNextLink(n)
				return title, link, err
			}
		}

	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if title == "" || link == "" {
			title, link, err = parseNode(c)
		}
	}

	return title, link, err
}

func parseNextLink(n *html.Node) (title, link string, err error) {

	if n.Type == html.ElementNode && n.Data == "a" {
		for _, v := range n.Attr {
			if v.Key == "href" {
				link = v.Val
			}

			if v.Key == "title" {
				title = v.Val
			}

			if title != "" && link != "" {
				return title, link, nil
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {

		if title == "" || link == "" {
			title, link, err = parseNextLink(c)
		}
	}

	if title == "" || link == "" {
		return title, link, errors.New("Failed to parse title or link")
	}
	return title, link, nil

}

// OldParseData is used in conjuction with parseYouTubeDataTokenizer
// NOT USED ANYMORE
func (y YoutubeParser) OldParseData(res *http.Response) (link, title string, err error) {
	newLink, newTitle, err := parseYoutubeDataTokenizer(res)
	return newLink, newTitle, err
}

// parseYouTubeDataTokenizer parses youtube website using tokenizer
func parseYoutubeDataTokenizer(res *http.Response) (link, title string, err error) {
	needTitle := false
	tokenizer := html.NewTokenizerFragment(res.Body, `<div>`)

	for {
		tempTag := tokenizer.Next()
		switch {
		case tempTag == html.ErrorToken:
			return "", "", errors.New("EOF")
		case tempTag == html.StartTagToken:
			tag := tokenizer.Token()

			isAnchor := tag.Data == "a"
			if isAnchor {
				for _, a := range tag.Attr {
					if a.Key == "href" {
						if matched, _ := regexp.MatchString(`/watch\?v=\w+`, a.Val); matched {
							link = a.Val
							needTitle = true
						}
					}
					if needTitle == true {
						if a.Key == "title" {
							title = a.Val
							needTitle = false
							return link, title, nil
						}
					}
				}
			}
		}
	}
}
