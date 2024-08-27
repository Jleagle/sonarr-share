package main

import (
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/dgraph-io/ristretto"
)

const cacheKey = "cache"

var (
	sonarrHost = flag.String("sonarr-host", "sonarr", "Sonarr host")
	sonarrPort = flag.Int("sonarr-port", 8989, "Sonarr port")
	sonarrKey  = flag.String("sonarr-key", "", "Sonarr key")
	serveHost  = flag.String("serve-host", "0.0.0.0", "Serve host")
	servePort  = flag.Int("serve-port", 8990, "Serve port")

	templates = template.Must(template.ParseFiles("main.gohtml"))
)

func main() {

	flag.Parse()

	if *sonarrKey == "" {
		fmt.Println("sonarr-key is required")
		return
	}

	cache, err := ristretto.NewCache(&ristretto.Config{NumCounters: 10, MaxCost: 10, BufferItems: 64})
	if err != nil {
		panic(err)
	}

	http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {

		var b []byte

		value, found := cache.Get(cacheKey)
		if !found {

			//goland:noinspection HttpUrlsUsage
			resp, err := http.Get(fmt.Sprintf("http://%s:%d/api/v3/series?apikey=%s", *sonarrHost, *sonarrPort, *sonarrKey))
			if err != nil {
				fmt.Println(err)
				return
			}

			//goland:noinspection GoUnhandledErrorResult
			defer resp.Body.Close()

			b, err = io.ReadAll(resp.Body)
			if err != nil {
				fmt.Println(err)
				return
			}

			cache.SetWithTTL(cacheKey, string(b), 1, time.Hour)

		} else if val, ok := value.([]byte); ok {
			b = val
		} else {
			fmt.Println("cache value is not []byte")
			return
		}

		var shows []Show
		err = json.Unmarshal(b, &shows)
		if err != nil {
			fmt.Println(err)
			return
		}

		slices.SortFunc(shows, func(a, b Show) int {
			return cmp.Or(
				cmp.Compare(a.NextAiring.Unix(), b.NextAiring.Unix()),
				cmp.Compare(a.PreviousAiring.Unix(), b.PreviousAiring.Unix()),
				cmp.Compare(a.SortTitle, b.SortTitle),
			)
		})

		err = templates.ExecuteTemplate(w, "main.gohtml", Data{Shows: shows})
		if err != nil {
			fmt.Println(err)
			return
		}
	})

	addr := fmt.Sprintf("%s:%d", *serveHost, *servePort)
	fmt.Println("Listening on", addr)
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
}

type Data struct {
	Shows []Show
}

type Show struct {
	Title           string `json:"title"`
	AlternateTitles []struct {
		Title        string `json:"title"`
		SeasonNumber int    `json:"seasonNumber"`
		Comment      string `json:"comment,omitempty"`
	} `json:"alternateTitles"`
	SortTitle      string    `json:"sortTitle"`
	Status         string    `json:"status"`
	Ended          bool      `json:"ended"`
	Overview       string    `json:"overview"`
	PreviousAiring time.Time `json:"previousAiring,omitempty"`
	Network        string    `json:"network"`
	AirTime        string    `json:"airTime,omitempty"`
	Images         []struct {
		CoverType string `json:"coverType"`
		Url       string `json:"url"`
		RemoteUrl string `json:"remoteUrl"`
	} `json:"images"`
	OriginalLanguage struct {
		Id   int    `json:"id"`
		Name string `json:"name"`
	} `json:"originalLanguage"`
	Seasons []struct {
		SeasonNumber int  `json:"seasonNumber"`
		Monitored    bool `json:"monitored"`
		Statistics   struct {
			PreviousAiring    time.Time `json:"previousAiring,omitempty"`
			EpisodeFileCount  int       `json:"episodeFileCount"`
			EpisodeCount      int       `json:"episodeCount"`
			TotalEpisodeCount int       `json:"totalEpisodeCount"`
			SizeOnDisk        int64     `json:"sizeOnDisk"`
			ReleaseGroups     []string  `json:"releaseGroups"`
			PercentOfEpisodes float64   `json:"percentOfEpisodes"`
			NextAiring        time.Time `json:"nextAiring,omitempty"`
		} `json:"statistics"`
	} `json:"seasons"`
	Year              int           `json:"year"`
	Path              string        `json:"path"`
	QualityProfileID  int           `json:"qualityProfileId"`
	SeasonFolder      bool          `json:"seasonFolder"`
	Monitored         bool          `json:"monitored"`
	MonitorNewItems   string        `json:"monitorNewItems"`
	UseSceneNumbering bool          `json:"useSceneNumbering"`
	Runtime           int           `json:"runtime"`
	TvdbID            int           `json:"tvdbId"`
	TvRageID          int           `json:"tvRageId"`
	TvMazeID          int           `json:"tvMazeId"`
	TmdbID            int           `json:"tmdbId"`
	FirstAired        time.Time     `json:"firstAired,omitempty"`
	LastAired         time.Time     `json:"lastAired,omitempty"`
	SeriesType        string        `json:"seriesType"`
	CleanTitle        string        `json:"cleanTitle"`
	IMDBID            string        `json:"imdbId"`
	TitleSlug         string        `json:"titleSlug"`
	RootFolderPath    string        `json:"rootFolderPath"`
	Certification     string        `json:"certification,omitempty"`
	Genres            []string      `json:"genres"`
	Tags              []interface{} `json:"tags"`
	Added             time.Time     `json:"added"`
	Ratings           struct {
		Votes int     `json:"votes"`
		Value float64 `json:"value"`
	} `json:"ratings"`
	Statistics struct {
		SeasonCount       int      `json:"seasonCount"`
		EpisodeFileCount  int      `json:"episodeFileCount"`
		EpisodeCount      int      `json:"episodeCount"`
		TotalEpisodeCount int      `json:"totalEpisodeCount"`
		SizeOnDisk        int64    `json:"sizeOnDisk"`
		ReleaseGroups     []string `json:"releaseGroups"`
		PercentOfEpisodes float64  `json:"percentOfEpisodes"`
	} `json:"statistics"`
	LanguageProfileId int       `json:"languageProfileId"`
	ID                int       `json:"id"`
	NextAiring        time.Time `json:"nextAiring,omitempty"`
}

func (s Show) Next() string {
	if s.Ended {
		return "Ended"
	}
	if s.NextAiring.IsZero() {
		return ""
	}
	return s.NextAiring.Format("_2 Jan 2006")
}

func (s Show) Last() string {
	if s.PreviousAiring.IsZero() {
		return ""
	}
	return s.PreviousAiring.Format("_2 Jan 2006")
}

func (s Show) IMDB() string {
	return strconv.FormatFloat(float64(s.Ratings.Value)*10, 'f', 0, 64)
}

func (s Show) Poster() string {
	for _, image := range s.Images {
		if image.CoverType == "poster" {
			q := url.Values{}
			q.Add("url", image.RemoteUrl)
			q.Add("output", "webp")
			q.Add("h", "400")
			q.Add("q", "100")
			return "https://images.weserv.nl/?" + q.Encode()
		}
	}
	return "https://critics.io/img/movies/poster-placeholder.png"
}
