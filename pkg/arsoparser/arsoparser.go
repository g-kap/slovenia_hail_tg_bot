package arsoparser

import (
	"context"
	"errors"
	"golang.org/x/net/html"
	"maps"
	"net/http"
	"sync"
	"time"
	"tocadanes/pkg/regions"
)

const (
	regionsLink = "https://meteo.arso.gov.si/uploads/probase/www/warning/text/sl/warning_hp_latest.html"
	citiesLink  = "https://meteo.arso.gov.si/uploads/probase/www/warning/text/sl/warning_hp-c_latest.html"
)

func parse(link string) (map[string]int, error) {
	resp, err := http.Get(link)
	if err != nil {
		return nil, err
	}
	respBody, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}
	tableNode, found := findFirstNode(respBody, func(node *html.Node) bool {
		return node.Data == "table" && getAttr(node, "class") == "meteoSI-table"
	})
	if !found {
		return nil, errors.New("table node not found")
	}
	rows, found := findAllNodes(tableNode, func(node *html.Node) bool {
		return node.Data == "tr"
	})
	if !found {
		return nil, errors.New("tr nodes not found")
	}
	result := make(map[string]int)
	for _, tr := range rows {
		cols, found := findAllNodes(tr, func(node *html.Node) bool {
			return node.Data == "td"
		})
		if !found {
			continue
		}
		if len(cols) != 5 {
			continue
		}
		regionName, found := findFirstNode(cols[0], func(node *html.Node) bool {
			return node.Type == html.TextNode && regions.IsSupportedRegion(node.Data)
		})
		if !found {
			continue
		}
		probability, found := findFirstNode(cols[2], func(node *html.Node) bool {
			return node.Type == html.TextNode && parseProbability(node.Data) != -1
		})
		if !found {
			continue
		}
		result[regionName.Data] = parseProbability(probability.Data)
	}

	return result, nil
}

func findFirstNode(doc *html.Node, checker func(*html.Node) bool) (*html.Node, bool) {
	if doc == nil {
		return nil, false
	}
	nodes, found := findAllNodes(doc, checker)
	if found && len(nodes) >= 1 {
		return nodes[0], true
	}
	return nil, false
}

func findAllNodes(doc *html.Node, checker func(*html.Node) bool) ([]*html.Node, bool) {
	if doc == nil {
		return nil, false
	}
	var foundNodes []*html.Node
	var crawler func(*html.Node)
	crawler = func(node *html.Node) {
		if checker(node) {
			foundNodes = append(foundNodes, node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			crawler(child)
		}
	}
	crawler(doc)
	return foundNodes, len(foundNodes) != 0
}

func getAttr(node *html.Node, attrName string) string {
	if node == nil {
		return ""
	}
	for _, attr := range node.Attr {
		if attr.Key == attrName {
			return attr.Val
		}
	}
	return ""
}

func parseProbability(s string) int {
	switch s {
	case "0/3":
		return 0
	case "1/3":
		return 1
	case "2/3":
		return 2
	case "3/3":
		return 3
	default:
		return -1
	}
}

type parser struct {
	prevState  map[string]int
	lastState  map[string]int
	interval   time.Duration
	mu         *sync.RWMutex
	log        Logger
	notifyChan chan struct{}
}

type Logger interface {
	InfoContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

func (p *parser) runOnce(ctx context.Context) (bool, error) {
	newState, err := parse(regionsLink)
	if err != nil {
		return false, err
	}
	cities, err := parse(citiesLink)
	if err != nil {
		return false, err
	}
	maps.Copy(newState, cities)
	p.mu.Lock()
	defer p.mu.Unlock()
	diff := MakeDiff(p.lastState, newState)
	if len(diff) > 0 {
		p.prevState = p.lastState
		p.lastState = newState
		return true, nil
	}
	return false, nil
}

func (p *parser) Run(ctx context.Context, done func()) {
	defer done()
	defer close(p.notifyChan)
	t := time.NewTimer(p.interval)
	for {
		select {
		case <-t.C:
			hasChanges, err := p.runOnce(ctx)
			if err != nil {
				p.log.ErrorContext(ctx, "error while updating state", err)
			}
			if hasChanges {
				p.notifyChan <- struct{}{}
			}
			t.Reset(p.interval)
		case <-ctx.Done():
			p.log.InfoContext(ctx, "terminating loop: context canceled")
			return
		}
	}
}

func MakeDiff(old, new map[string]int) map[string]int {
	res := make(map[string]int)
	for k := range new {
		if old[k] != new[k] {
			res[k] = new[k] - old[k]
		}
	}
	return res
}

func (p *parser) LastState() map[string]int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return maps.Clone(p.lastState)
}

func (p *parser) PrevState() map[string]int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return maps.Clone(p.prevState)
}

func (p *parser) Changes() <-chan struct{} {
	return p.notifyChan
}

type Parser interface {
	LastState() map[string]int
	PrevState() map[string]int
	Changes() <-chan struct{}
	Run(ctx context.Context, done func())
}

func NewParser(log Logger) Parser {
	return &parser{
		interval:   time.Second * 10,
		mu:         &sync.RWMutex{},
		log:        log,
		notifyChan: make(chan struct{}),
	}
}
