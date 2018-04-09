package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/david-ds/learn-grpc/todo"
	"github.com/golang/protobuf/proto"
	context "golang.org/x/net/context"
	"google.golang.org/grpc"
)

type todoServer struct {
}

const dbFilePath = "todo.db"

func main() {
	var srv todoServer

	conn, err := net.Listen("tcp", "127.0.0.1:8888")
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not connect to port 8888 : %v", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(grpc.UnaryInterceptor(logInterceptor))

	todo.RegisterAddServiceServer(grpcServer, &srv)
	todo.RegisterListServiceServer(grpcServer, &srv)
	todo.RegisterDoneServiceServer(grpcServer, &srv)
	todo.RegisterDropServiceServer(grpcServer, &srv)

	if err := grpcServer.Serve(conn); err != nil {
		fmt.Fprintf(os.Stderr, "could not start the grpc server : %v", err)
		os.Exit(1)
	}
}

func (s todoServer) Add(ctx context.Context, text *todo.Text) (*todo.Task, error) {
	task := &todo.Task{Title: text.Text, Done: false}

	f, err := os.OpenFile(dbFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("could not open db file : %v", err)
	}

	defer f.Close()

	b, err := proto.Marshal(task)
	if err != nil {
		return nil, fmt.Errorf("could not serialize task : %v", err)
	}

	length := int64(len(b))
	err = binary.Write(f, binary.LittleEndian, length)
	if err != nil {
		return nil, fmt.Errorf("could not write length of the task : %v", err)
	}

	_, err = f.Write(b)
	if err != nil {
		return nil, fmt.Errorf("could not write task to the db : %v", err)
	}

	return task, nil
}

func (s todoServer) List(ctx context.Context, void *todo.Void) (*todo.TaskList, error) {
	taskList := &todo.TaskList{}

	f, err := os.Open(dbFilePath)
	if os.IsNotExist(err) {
		return taskList, nil
	} else if err != nil {
		return nil, fmt.Errorf("could not read db file : %v", err)
	}

	defer f.Close()

	var length int64
	for {
		if err = binary.Read(f, binary.LittleEndian, &length); err == io.EOF {
			return taskList, nil
		} else if err != nil {
			return nil, fmt.Errorf("could not read length of the task : %v", err)
		}

		b := make([]byte, length)
		if _, err = f.Read(b); err != nil {
			return nil, fmt.Errorf("could not read task : %v", err)
		}

		var task todo.Task
		if err = proto.Unmarshal(b, &task); err != nil {
			return nil, fmt.Errorf("could not deserialize task : %v", err)
		}

		taskList.Tasks = append(taskList.Tasks, &task)
	}
}

func setTasks(ctx context.Context, tasks *todo.TaskList) error {
	f, err := os.OpenFile(dbFilePath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("could not open db file : %v", err)
	}

	for _, task := range tasks.Tasks {
		b, err := proto.Marshal(task)
		if err != nil {
			return fmt.Errorf("could not serialize task : %v", err)
		}

		l := int64(len(b))
		err = binary.Write(f, binary.LittleEndian, l)
		if err != nil {
			return fmt.Errorf("could not write length of the task : %v", err)
		}

		_, err = f.Write(b)
		if err != nil {
			return fmt.Errorf("could not write the task : %v", err)
		}
	}

	return nil
}

func (s todoServer) Done(ctx context.Context, text *todo.Text) (*todo.TaskList, error) {
	allTasks, err := s.List(ctx, &todo.Void{})
	if err != nil {
		return nil, fmt.Errorf("could not load the tasks : %v", err)
	}

	updatedTasks := &todo.TaskList{}

	for i, task := range allTasks.Tasks {
		if task.Title == text.Text {
			allTasks.Tasks[i].Done = true
			updatedTasks.Tasks = append(updatedTasks.Tasks, task)
		}
	}

	if err := setTasks(ctx, allTasks); err != nil {
		return nil, fmt.Errorf("could not write tasks : %v", err)
	}

	return updatedTasks, nil
}

func (s todoServer) Drop(ctx context.Context, void *todo.Void) (*todo.Void, error) {
	if err := os.Remove(dbFilePath); os.IsNotExist(err) {
		return &todo.Void{}, nil
	} else if err != nil {
		return nil, fmt.Errorf("could not drop database : %v", err)
	}

	return &todo.Void{}, nil
}

// logInterceptor logs the duration of the requests
func logInterceptor(ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	h, err := handler(ctx, req)

	duration := time.Since(start)

	log.Printf("request duration %s", duration)

	return h, err
}
