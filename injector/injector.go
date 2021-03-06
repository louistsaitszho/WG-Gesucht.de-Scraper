// Package injector contains a series of functions to inject all the fields of an Ad.
// Right now this is only target to support WG-Gesucht.de
package injector

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

//offer has to be here because "extensino function" is not a thing in go

//1. will have offer and request (both as Ad), and list is type that contains List<Ad>
//2. will use a lot of interfaces to share function, e.g. things that "HasID" has getter and setter for ID
//3. maybe emmbed will be the best solution, but right now I am going with interface, because I am sure that is gonna work accordingly

//just extract everything from offer and request, and then see what can be moved to Ad

//Ad can be offer or request, and it only contains items that is shared between both of them
type Ad interface {
	AdID() int
	SetAdID(id int)
	Title() string
	SetTitle(title string)
	Name() string
	SetName(name string)
	IsActive() bool
	SetIsActive(active bool)
	//TODO not sure what type a date should be
	// MemberSince() string
	// setMemberSince(ms string)

	//last online
}

//List of Ad
type List interface {
	Page() int
	SetPage(page int)
}

//Offer = that person has a place to rent out
type Offer struct {
	adID             int
	title            string
	name             string
	active           bool
	address          string
	BaseRent         int
	Utilities        int
	Miscellaneous    int
	Deposit          int
	ExistingEquipFee int
	RoomSize         int
	TotalSize        int
	FlatmateAgeMin   int
	FlatmateAgeMax   int
}

func (o Offer) String() string {
	text, _ := json.Marshal(o)
	return string(text)
}

//Request = that person want to rent a place
type Request struct {
	adID  int
	title string
	name  string
}

//AdID is basicallya getter for adID in Offer
func (o *Offer) AdID() int {
	return o.adID
}

//SetAdID sets the id of the offer
func (o *Offer) SetAdID(id int) {
	o.adID = id
}

//Title returns the title of the offer
func (o *Offer) Title() string {
	return o.title
}

//SetTitle sets the title of the offer
func (o *Offer) SetTitle(title string) {
	o.title = title
}

//Name return the poster name of the offer
func (o *Offer) Name() string {
	return o.name
}

//SetName sets the name of the poster of the offer
func (o *Offer) SetName(name string) {
	o.name = name
}

//IsActive return if the offer is active
func (o *Offer) IsActive() bool {
	return o.active
}

//SetIsActive set if the offer is active or not
func (o *Offer) SetIsActive(active bool) {
	o.active = active
}

//InjectAdID put the AdID into the right place, or return an error
//if it cannot find it
//This function is uses InjectActiveness to inject ID correctly
//TODO handle deactivited ad: https://www.wg-gesucht.de/en/wg-zimmer-in-Berlin-Kreuzberg.6475694.html
//TODO sometimes they hide the picture, and it screw up everything
func InjectAdID(ad Ad, doc *goquery.Document) (Ad, error) {
	ad, errActiveness := InjectActiveness(ad, doc)
	if errActiveness != nil {
		return ad, errActiveness
	}

	if ad.IsActive() { //extract ID from visible content
		outside := doc.Find("div#main_content").Find("div#main_column").Find(".panel.panel-default").Find(".panel-body").Find(".row").Find(".col-xs-12").Find(".row").Find(".hidden-xs.hidden-sm").Find(".col-md-4").Find(".row").Find(".col-md-12").Slice(1, 2)
		garbage := outside.Children()
		idString := strings.Replace(strings.Replace(strings.Replace(outside.Text(), garbage.Text(), "", -1), "\n", "", -1), " ", "", -1)

		id, err := strconv.Atoi(idString)
		if err != nil {
			return ad, err
		}
		ad.SetAdID(id)
	} else { //extract ID from urls in content
		filteredList := doc.Find("link").FilterFunction(func(i int, s *goquery.Selection) bool {
			// canonical
			rel, exists := s.Attr("rel")
			if exists {
				if rel == "canonical" {
					return true
				}
			}
			return false
		})

		if filteredList.Length() != 1 {
			return ad, errors.New("There are more than 1 canonical for some reason")
		}

		href, existsHref := filteredList.Attr("href")
		if existsHref {
			println("href = " + href)
			splited := strings.Split(href, ".")
			idString := splited[len(splited)-2]
			id, err := strconv.Atoi(idString)
			if err != nil {
				return ad, err
			}
			ad.SetAdID(id)
		} else {
			return ad, errors.New("Failed to find AdID in href from canonical")
		}
	}
	return ad, nil
}

//InjectAdTitle put the title of the ad into the right place
func InjectAdTitle(ad Ad, doc *goquery.Document) (Ad, error) {
	ad, errActiveness := InjectActiveness(ad, doc)
	if errActiveness != nil {
		return ad, errActiveness
	}

	if ad.IsActive() {
		outside := doc.Find("div#main_content").Find("div#main_column").Find(".panel.panel-default").Find(".panel-body").Find("div.noprint.showOnGalleryOnly").Find("h1#sliderTopTitle")
		garbage := outside.Children()
		title := strings.TrimSpace(strings.Replace(strings.Replace(outside.Text(), garbage.Text(), "", -1), "\n", "", -1))
		if len(title) == 0 {
			return ad, errors.New("Cannot find title (length == 0)")
		}
		ad.SetTitle(title)
		return ad, nil
	}
	title := strings.TrimSpace(doc.Find("h1#sliderTopTitle.headline.headline-detailed-view-title").Text())
	if len(title) == 0 {
		return ad, errors.New("Cannot find title (length == 0) (deactive)")
	}
	ad.SetTitle(title)
	return ad, nil
}

//InjectActiveness inject if the ad is active or not. Note that this does not care if the link is actually
//an Ad. That is not the duty of this function, and if you do that, it may return wrong result (in this case,
//return that the ad is active)
func InjectActiveness(ad Ad, doc *goquery.Document) (Ad, error) {
	deactivedBlock := doc.Find(".panel.panel-deactivated").Text() //if this exist, the ad is not active
	ad.SetIsActive(!(len(deactivedBlock) > 0))
	return ad, nil
}

//InjectOfferActiveness is the same as InjectActiveness but takes Offer instead and do the type assertion myself.
func InjectOfferActiveness(offer Offer, doc *goquery.Document) (Offer, error) {
	offerAd, error := InjectActiveness(&offer, doc)
	offerOffer := offerAd.(*Offer)
	return *offerOffer, error
}

//InjectAddress inject the address of an offer.
func InjectAddress(offer Offer, doc *goquery.Document) (Offer, error) {
	address := doc.Find("div#main_content.row").Find("div#main_column.col-md-8").Find("div.panel-body").Find("div.row").Find("div.col-sm-4.mb10").Find("a")
	trimFullAddress := strings.TrimSpace(strings.Trim(address.Text(), "\n"))
	lines := strings.Split(trimFullAddress, "\n")
	fmt.Printf("line length = %d\n", len(lines))
	var cleanLines []string
	for i, line := range lines {
		cleanThisLine := strings.TrimSpace(line)
		if len(cleanThisLine) > 0 {
			fmt.Printf("This is line %d: %s\n", i, cleanThisLine)
			cleanLines = append(cleanLines, cleanThisLine)
		} else {
			fmt.Printf("Line %d got thrown away\n", i)
		}
	}
	offer.address = strings.Join(cleanLines, ", ")
	// offer.address = strings.TrimSpace(strings.Replace(strings.Replace(address.Text(), "  ", "", -1), "\n\n\n", ", ", -1))

	return offer, nil
}

// func (offer *Offer) injectDescription(doc *goquery.Document) error {
// 	// doc.Find("div.freitext").Each(func(index int, item *goquery.Selection) {
// 	// 	//fmt.Printf("Description:\n%s\n", strings.TrimSpace(item.Text()))
// 	// 	thisOffer.description = strings.TrimSpace(item.Text())
// 	// })
// }

// //injectArea extract the the areas from doc and put it into offer. It returns an error if:
// //1) Multiple conflicting area found
// //2) Room area is larger than house area
// //3) Room area not found
// func (offer *Offer) injectArea(doc *goquery.Document) error {
// 	// doc.Find("#rent_wrapper").Find("label.amount").Each(func(index int, item *goquery.Selection) {
// 	// 	fmt.Printf("area[%d] = %s\n", index, strings.TrimSpace(item.Text()))
// 	// })
// }

// //injectAddress extracts all the address stuff from the page:
// //1. Address in text
// //2. Is the address just an approximation
// //3. Latitude (from onClick loagGMap)
// //4. Longitude (from onClick loagGMap)
// func (offer *Offer) injectAddress(doc *goquery.Document) error {
// 	// doc.Find("#main_content").Find("#main_column").Find(".panel-body").Find(".col-sm-4").Find("[onclick]").Each(func(index int, item *goquery.Selection) {
// 	// 	fmt.Printf("address = %s\n", strings.TrimSpace(item.Text()))
// 	// })
// }

// //injectPrices extract the prices from doc and put it into offer. It returns an error if:
// //1) Multiple conflicting prices found
// //2) no prices found
// //It will try to fetch all the sub-prices if possible
// func (offer *Offer) injectPrices(doc *goquery.Document) error {
// 	// doc.Find("#graph_wrapper").Find(".basic_facts_bottom_part").Find(".amount").Each(func(index int, item *goquery.Selection) {
// 	// 	// fmt.Printf("rent = %s\n", strings.TrimSpace(item.Text()))
// 	// 	// *offer.price = int(item.Text())	//TODO somthing like this probably
// 	// })
// }

// //injectAvailability extract the start date and end date of the offer into AvailableFrom and AvailableTo
// func (offer *Offer) injectAvailability(doc *goquery.Document) error {

// }

// //injectCurrentOccupantSize extracts the info of the current occupant(s), i.e. OccupantsMale, and OccupantsFemale
// func (offer *Offer) injectCurrentOccupantSize(doc *goquery.Document) error {
// 	//ul-detailed-view-datasheet print_text_left
// }

// //injectCurrentOccupantSize extracts the info of the current occupant(s), i.e. OccupantsAgeMin, and OccupantsAgeMax
// func (offer *Offer) injectCurrentOccupantAge(doc *goquery.Document) error {
// 	//ul-detailed-view-datasheet print_text_left
// }

// //injectTargetLimition extracts
// func (offer *Offer) injectTargetLimition(doc *goquery.Document) error {
// 	//ul-detailed-view-datasheet print_text_left
// }

// //injectLanguages extracts the list of languages that the offer has listed as the languages that he/she/they speak(s)
// //TODO make a map that turns flag title back to standard language code. Also, check if golang has standard language code
// func (offer *Offer) injectLanguages(doc *goquery.Document) error {
// 	// doc.Find("img.flgS").Each(func(index int, item *goquery.Selection) {
// 	// 	title, _ := item.Attr("title")
// 	// 	fmt.Printf("flag %d = %s\n", index, title)
// 	// })
// }

// //injectImages extracts the list of images urls (if available)
// func (offer *Offer) injectImages(doc *goquery.Document) error {

// }

// //injectMinorDetails basically runs through the 3 column grid and put them into
// //their corresponding field in the offer struct
// func (offer *Offer) injectMinorDetails(doc *goquery.Document) error {
// 	//pretty sure the .Find() does not that multiple class that way
// 	doc.Find(".col-xs-6 .col-sm-4 .col-md-4 .print_text_left").Each(func(index int, item *goquery.Selection) {
// 		classes, _ := item.Find("span").First().Attr("class") //this should give you "glyphicons glyphicons-?????? noprint"
// 		text = item.Text()
// 		switch classes {
// 		case "glyphicons glyphicons-folder-closed noprint": //e.g. "Washing machine, Balcony, Basement/Cellar, Elevator, Pets are welcome "
// 			//-> washing machine
// 			//-> elevator
// 			//-> balcony
// 			//-> pet
// 			//-> basement
// 		case "glyphicons glyphicons-car noprint": //e.g. "Many free parking lots"
// 		case "glyphicons glyphicons-fire noprint": //e.g. "Central heating"
// 			//-> heating
// 		case "glyphicons glyphicons-fabric noprint": //e.g. "Polished floorboards"
// 		case "glyphicons glyphicons-wifi-alt noprint": //e.g. "DSL, WLAN 26-50 Mbit/s"
// 			//-> internet
// 		case "glyphicons glyphicons-bath-bathtub noprint": //e.g. "Bathtub"
// 			//-> bathtub
// 		case "glyphicons glyphicons-bed noprint": //e.g. " 3rd floor, furnished " (Yes, extra spaces included -_-)
// 			//-> floor
// 			//-> furnished

// 			//1. string to []string by commar
// 			//2. trim
// 			//3. regex "??? floor" or other form of levels
// 			//4. furnished, partically furnished, or not furnished
// 		case "glyphicons glyphicons-building noprint": //e.g. "Industrial building"
// 			//might want to put default here to catch other stuff
// 		}

// 		//telephone
// 		//flooring
// 		//tv
// 		//dishwasher
// 		//terrace
// 		//garden
// 		//shared garden
// 		//bicycle storage
// 		//green power
// 	})
// 	return nil
// }

// //injectPosterName extract the (user)name of the person who post this Offer/Request, i.e. the "Name: " field
// func (offer *Offer) injectPosterName(doc *goquery.Document) error {

// }
