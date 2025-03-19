# API Client for Hacker News 

Links:
* [API Docs](https://github.com/HackerNews/API)
* [Hacker News](https://news.ycombinator.com/)

## Features:

* Retrieve user data (comments, stories, etc.)
* Fetch recent updates and news (Best, New, etc.)
* Filter and sort the results by multiple fields

## Installation

```sh
go get -u github.com/imotkin/hn-client
```

## Examples 

#### Import the library and create a client

```go
import (
    "github.com/imotkin/hn-client"
)

func main() {
    client := hn.NewClient(nil)
    
    // Set a limit on the number of workers
    hn.SetMaxWorkers(10)

    ctx := context.Background()
}
```

#### Fetch user data (comments, stories, or custom items)

```go
comments, _ := client.Users.Comments(ctx, "johndoe")

stories, _ := client.Users.Stories(ctx, "johndoe")

items, _ := client.Users.Items(ctx, "johndoe", func (i hn.Item) bool {
    return i.Type == hn.CommentType && strings.Contains(i.Title, "Go")
})
```

#### Fetch a list of items with a filter

```go
items, _ := client.Items.List(ctx, []uint{1, 2, 3}, func (i hn.Item) bool {
    return i.Type == hn.CommentType && strings.Contains(i.Title, "Go")
})

// Convert to a slice of comments
comments := hn.ToList[hn.Comment](items)

for _, comment := range comments {
    fmt.Println(comment.ID, comment.Text)
}
```

#### Sort items by predefined fields (id, time, score, or type)

```go
hn.SortScore(comments, hn.Ascending)

hn.SortTime(stories, hn.Descending)
```

#### Sort items by your custom order

```go
hn.Sort(comments, func(a, b hn.Comment) int {
    return cmp.Compare(a.Text, b.Text)
})

hn.Sort(stories, func(a, b hn.Story) int {
    return cmp.Compare(a.By, b.By)
})
```

#### Fetch recent updates and news

```go
recent, _ := client.Live.Recent(context.Background(), 10)

updates, _ := client.Live.UpdateList(ctx, nil)

best, _ := client.Live.BestList(ctx, func (i hn.Item) bool {
    return strings.Contains(i.Title, "Go")
})
```