package parsers

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestParseYoutubeData(t *testing.T) {

	t.Run("ParseYoutubeDataTokenizer from response_test.dat", func(t *testing.T) {
		file, err := os.Open("response_test.dat")
		if err != nil {
			t.Errorf("Failed to open file with test data; reason: %s", err)
		}
		body, err := ioutil.ReadAll(file)
		if err != nil {
			t.Errorf("Failed to read test data from file; reason: %s", err)
		}
		server := makeFakeYoutubeServer(body)
		defer server.Close()

		client := http.Client{}
		res, _ := client.Get(server.URL)
		defer res.Body.Close()
		wantLink := "/watch?v=KR-eV7fHNbM"
		wantTitle := "TheFatRat - The Calling (feat. Laura Brehm)"
		gotLink, gotTitle, err := parseYoutubeDataTokenizer(res)
		if err != nil {
			t.Errorf("Failed to parse response body; %s; reason: %s", res.Body, err)
		}

		assertLinkEquals(t, wantLink, gotLink)
		assertTitleEquals(t, wantTitle, gotTitle)

	})

	t.Run("ParseYoutubeDataTokenizer from response2_test.dat", func(t *testing.T) {

		file, err := os.Open("response2_test.dat")
		if err != nil {
			t.Errorf("Failed to open file with test data; reason: %s", err)
		}
		body, err := ioutil.ReadAll(file)
		if err != nil {
			t.Errorf("Failed to read test data from file; reason: %s", err)
		}
		server := makeFakeYoutubeServer(body)
		defer server.Close()

		client := http.Client{}
		res, _ := client.Get(server.URL)
		defer res.Body.Close()
		wantLink := "/watch?v=TsTFVdcpLrE"
		wantTitle := "Hans Zimmer - Time ( Cyberdesign Remix )"
		gotLink, gotTitle, err := parseYoutubeDataTokenizer(res)
		if err != nil {
			t.Errorf("Failed to parse response body; %s; reason: %s", res.Body, err)
		}

		assertLinkEquals(t, wantLink, gotLink)
		assertTitleEquals(t, wantTitle, gotTitle)
	})
}

func TestNewParseYoutubeData(t *testing.T) {
	y := YoutubeParser{
		Log: logrus.New(),
	}

	y.Log.Out = ioutil.Discard
	t.Run("YouTube NextLink parser from reponse_test.dat", func(t *testing.T) {

		file, err := os.Open("response_test.dat")
		if err != nil {
			t.Errorf("Failed to open file with test data; reason: %s", err)
		}
		body, err := ioutil.ReadAll(file)
		if err != nil {
			t.Errorf("Failed to read test data from file; reason: %s", err)
		}
		server := makeFakeYoutubeServer(body)
		defer server.Close()

		client := http.Client{}
		res, _ := client.Get(server.URL)
		defer res.Body.Close()
		wantLink := "/watch?v=KR-eV7fHNbM"
		wantTitle := "TheFatRat - The Calling (feat. Laura Brehm)"
		gotLink, gotTitle, err := y.ParseData(res)
		if err != nil {
			t.Errorf("Failed to parse response body; reason: %s", err)
		}

		assertLinkEquals(t, wantLink, gotLink)
		assertTitleEquals(t, wantTitle, gotTitle)

	})
}

func makeFakeYoutubeServer(body []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(body)

	}))
}

func assertLinkEquals(t *testing.T, wantLink, gotLink string) {
	t.Helper()
	if wantLink != gotLink {
		t.Errorf("Got %s, want %s", gotLink, wantLink)
	}
}

func assertTitleEquals(t *testing.T, wantTitle, gotTitle string) {
	t.Helper()
	if wantTitle != gotTitle {
		t.Errorf("Got %s, want %s", gotTitle, wantTitle)
	}
}
