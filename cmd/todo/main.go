package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/david-ds/learn-grpc/todo"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
)

func main() {

	if len(os.Args) <= 1 {
		fmt.Fprintf(os.Stderr, "not enough arguments")
		os.Exit(1)
	}

	conn, err := grpc.Dial("127.0.0.1:8888", grpc.WithInsecure())
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to contact the server : %v", err)
		os.Exit(1)
	}

	defer conn.Close()

	addClient := todo.NewAddServiceClient(conn)
	listClient := todo.NewListServiceClient(conn)
	doneClient := todo.NewDoneServiceClient(conn)
	dropClient := todo.NewDropServiceClient(conn)

	switch command := os.Args[1]; command {
	case "add":
		err = add(addClient, os.Args[2])
		if err != nil {
			log.Fatalf("could not add task : %v", err)
		}
	case "list":
		err = list(listClient)
		if err != nil {
			log.Fatalf("could not list tasks : %v", err)
		}
	case "done":
		err = done(doneClient, os.Args[2])
		if err != nil {
			log.Fatalf("could not mark tasks as done : %v", err)
		}
	case "drop":
		err = drop(dropClient)
		if err != nil {
			log.Fatalf("could not drop database : %v", err)
		}
	default:
		log.Fatalf("unknown command : %s", command)
	}

}

func add(client todo.AddServiceClient, text string) error {
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Second)
	task, err := client.Add(ctx, &todo.Text{Text: text})
	if err != nil {
		return fmt.Errorf("could not add task : %v", err)
	}

	fmt.Printf("task %s added\n", task.Title)
	return nil
}

func list(client todo.ListServiceClient) error {
	taskList, err := client.List(context.Background(), &todo.Void{})
	if err != nil {
		return fmt.Errorf("could not list tasks : %v", err)
	}

	for _, task := range taskList.Tasks {
		if task.Done {
			fmt.Printf("✅ ")
		} else {
			fmt.Printf("  ")
		}
		fmt.Printf(" %s\n", task.Title)
	}

	return nil
}

func done(client todo.DoneServiceClient, query string) error {
	taskList, err := client.Done(context.Background(), &todo.Text{Text: query})
	if err != nil {
		return fmt.Errorf("could not update tasks : %v", err)
	}

	fmt.Printf("%d tasks have been updated\n", len(taskList.Tasks))
	return nil
}

func drop(client todo.DropServiceClient) error {
	if _, err := client.Drop(context.Background(), &todo.Void{}); err != nil {
		return fmt.Errorf("could not drop database : %v", err)
	}

	fmt.Println("The tasks have been successfuly dropped")
	return nil
}
