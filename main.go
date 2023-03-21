package main

import (
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	"sort"
	"sync"
	"errors"
	"github.com/gophercises/quiet_hn/hn" 
	// install using go mod init test.go/packages
	// then go mod tidy
	// then go install
)

func main() {
	// parse command line flags
	var port, numStories int

	flag.IntVar(&port, "port", 3000, "the port to start the web server on")
	flag.IntVar(&numStories, "num_stories", 30, "the number of top stories to display")
	flag.Parse()

	tpl := template.Must(template.ParseFiles("./index.gohtml"))

	http.HandleFunc("/", handler(numStories, tpl))

	// Start the server
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}

func handler(numStories int, tpl *template.Template) http.HandlerFunc {

	//BONUS improved cache:
	sc := storyCache{
		numStories: numStories,
		duration: 6 * time.Second,
	}
	
	go func() { //this function updates the cache every 3 seconds, blocking requests to the routine using mutex by locking the cache
		ticker := time.NewTicker(3 * time.Second)
		for {
			temp := storyCache{
				numStories: numStories,
				duration: 6 * time.Second,
			}
			temp.stories()
			sc.mutex.Lock()		
			sc.cache = temp.cache
			sc.expiration = temp.expiration
			sc.mutex.Unlock()
			<-ticker.C //run the ticker last so the above block runs before the ticker runs a check
		}
	}()

	//end bonus

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		///////////////
		// stories, err := getCachedStories(numStories); //use cache function - take 4
		stories, err := sc.stories() //Bonus take -- improved cache

		if err != nil{
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		///////////////
		///////////////

		data := templateData{
			Stories: stories,
			Time:    time.Now().Sub(start),
		}
		err = tpl.Execute(w, data)
		if err != nil {
			http.Error(w, "Failed to process the template", http.StatusInternalServerError)
			return
		}
	})
}

//TAKE 4&5 - ADD A CACHE AND MUTEX

var ( //declare cache vars
	cache 			[]item
	cacheExpiration time.Time
	cacheMutex sync.Mutex // create a mutex - take 5
)

func getCachedStories(numStories int) ([]item, error){
	cacheMutex.Lock() //lock a mutex, this causes the code below to only be accessed by 1 goroutine at a time, preventing race conditions

	// more info: this helps prevent calling getTopStories too many times, as another request with the same info would have to wait until the mutex
	// is unlocked, this means that the cache would be updated by the first call, and the second call wouldn't need to request getTopStories
	
	defer cacheMutex.Unlock() //it's good to defer the unlock, we guarantee that once the function runs, the mutex is always unlocked 
	
	//if cache is not expired yet
	if time.Now().Sub(cacheExpiration) < 0 {
		return cache, nil
	}

	//grab new stories
	stories, err := getTopStories(numStories)
	if err != nil {
		return nil, err
	}

	cache = stories //set new stories to cache
	cacheExpiration = time.Now().Add(1 * time.Second) //set new expiration date a minute from now

	return cache, nil //return cached stories
}




// BONUS - IMPROVE THE CACHE TO UPDATE BEFORE WE NEED A NEW ONE
type storyCache struct {
	numStories		int
	cache 			[]item
	expiration		time.Time
	duration		time.Duration
	mutex			sync.Mutex
}


func (sc *storyCache) stories() ([]item, error) { 
	//storyCache is passed as a pointer (*) so mutex's are always the same	
	sc.mutex.Lock() 
	defer sc.mutex.Unlock()

	//if cache is not expired yet
	if time.Now().Sub(sc.expiration) < 0 {
		return sc.cache, nil
	}

	//grab new stories
	stories, err := getTopStories(sc.numStories)
	if err != nil {
		return nil, err
	}

	sc.expiration = time.Now().Add(sc.duration) //set new expiration date a minute from now

	sc.cache = stories
	return sc.cache, nil

}



//Main function
func getTopStories(numStories int ) ( []item, error){
	var client hn.Client
		ids, err := client.TopItems()
		if err != nil {
			return nil, errors.New("Failed to load top stories");
		}
	



			//TAKE 1: 

			// var stories []item
			// for _, id := range ids {		
			// //make a result struct (object type)
			// type result struct {
			// 	item item
			// 	err error
			// }
			
			// //make a channel of results (a independent concurrent channel)
			// resultCh := make(chan result)
			// // 
			// go func(id int) { //defines anonymous function and params
			// hnItem, err := client.GetItem(id) //ask for item
			// if err != nil {
			// 	resultCh <- result{err:err} //send error to channel on error
			// }
			// resultCh <- result{item: parseHNItem(hnItem)} //get hnItem and parse it, send to channel 
			// }(id)//pass id value as argument

			// //the above is now concurrent

			// res := <-resultCh; //this is blocking right now, which means that concurrency doesn't work yet, 
			// //i.e. it needs the goroutine to finish before it can continue
			
			// if res.err != nil { //check for errors
			// 	continue
			// }
			// if isStoryLink(res.item) { //otherwise, add it to the stories struct
			// 	stories = append(stories, res.item)
			// 	if len(stories) >= numStories {
			// 		break
			// 	}
			// }	
			// }		
			//END TAKE 1://////

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////


			// TAKE 2 ///
			
			// //make a result struct (object type)
			// type result struct {
			// 	idx int //create an index so we save the order and don't need to worry about order of routines 
			// 	item item
			// 	err error
			// }
			
			// resultCh := make(chan result) //make an entire channel of results			
			
			// // for index, id := range ids {  
			// 	// range ids is changed to a for loop so we don't create 400 routines 
				
			// for i := 0; i<numStories; i++ { // make a bunch of goroutines to grab each result
				
			// 	go func(index, id int) { 
			// 	hnItem, err := client.GetItem(id) //ask for item
			// 	if err != nil {
			// 		resultCh <- result{idx: index, err:err} //send error to channel on error
			// 	}
			// 	resultCh <- result{item: parseHNItem(hnItem)} //get hnItem and parse it, send to channel 
			// 	}(i, ids[i])//this line passes the argument to the anonymous function

			// }

			// var results []result //create a slice for all results
			// for i:= 0; i < numStories; i++ {
			// 	results = append(results, <-resultCh) //pull a result from result channel resultCh, goroutines work with one value at a time, so every time you use <-resultCh it pulls the next
			// }

			// sort.Slice(results, func(i int, j int) bool{ //use sort package so sort slice
			// 	return results[i].idx < results[j].idx //sort comparison (called a less function)
			// })

			// var stories []item //create a stories slice
			// for _, reslt := range results { //loop and validate stories, append useful stories
			// 	if reslt.err != nil {
			// 		continue
			// 	}
			// 	if(isStoryLink(reslt.item)){
			// 		stories = append(stories, reslt.item)
			// 	}

			// }
			// END TAKE 2		


			//// TAKE 3 - EXACTLY 30 STORIES
				// stories:= getStories(ids[0:numStories])// would pull 30, but we need a better answer
				var stories []item
				at:= 0
				
				for len(stories) < numStories{
					need := (numStories - len(stories)) * 5 / 4 //grab more than we need to allow for fewer retries

					stories = append(stories, getStories(ids[at:at+need])...) 
					//we append with a varying range of ids (we don't know if it'll be 30 or not) 
					//so it's called  'variadic' parameters, we can use an ellipses (...) to allow the function to use all the parameters
					//without needing a strong definition of the parameter length
					
					at+=need //set the index to the number of appended stories, that way we can fetch more if we only fetched, say, 25 stories
				}


			/////END TAKE 3

			/////


		return stories[:numStories], nil //finally return 30 stories and no error

}

//TAKE 3 -- BREAK OUT STORY FETCH TO AN INDEPENDENT FUNCTION SO WE CAN USE THIS RECURSIVELY TO GRAB 30 VALID STORIES
func getStories(ids []int) []item{ //get slice of ids, return slice of items 



		//make a result struct (object type)
		type result struct {
			idx int //create an index so we save the order and don't need to worry about order of routines 
			item item
			err error
		}
		
		resultCh := make(chan result) //make an entire channel of results			
			
		for i := 0; i<len(ids); i++ { // make a bunch of goroutines to grab each result
			
			var client hn.Client //create a client, race-condition free since the loop is constantly calling two goroutines (== check and setting c.apiBase -- line 23/24) we need to initiate the client each time we call it instead, this is ok since the constructor is so simple

			go func(index, id int) { 
			hnItem, err := client.GetItem(id) //ask for item
			if err != nil {
				resultCh <- result{idx: index, err:err} //send error to channel on error
			}
			resultCh <- result{item: parseHNItem(hnItem)} //get hnItem and parse it, send to channel 
			}(i, ids[i])//this line passes the argument to the anonymous function

		}

		var results []result //create a slice for all results
		for i:= 0; i < len(ids); i++ {
			results = append(results, <-resultCh) //pull a result from result channel resultCh, goroutines work with one value at a time, so every time you use <-resultCh it pulls the next
		}

		sort.Slice(results, func(i int, j int) bool{ //use sort package so sort slice
			return results[i].idx < results[j].idx //sort comparison (called a less function)
		})

		var stories []item //create a stories slice
		for _, reslt := range results { //loop and validate stories, append useful stories
			if reslt.err != nil {
				continue
			}
			if(isStoryLink(reslt.item)){
				stories = append(stories, reslt.item)
			}

		}
		return stories
}
 

func isStoryLink(item item) bool {
	return item.Type == "story" && item.URL != ""
}

func parseHNItem(hnItem hn.Item) item {
	ret := item{Item: hnItem}
	url, err := url.Parse(ret.URL)
	if err == nil {
		ret.Host = strings.TrimPrefix(url.Hostname(), "www.")
	}
	return ret
}

// item is the same as the hn.Item, but adds the Host field
type item struct {
	hn.Item
	Host string
}

type templateData struct {
	Stories []item
	Time    time.Duration
}
