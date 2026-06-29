package pipeline

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPipelineContext_SetGet(t *testing.T) {
	c := NewPipelineContext("test")
	c.Set("a", 1)
	v, ok := c.Get("a")
	if !ok || v.(int) != 1 {
		t.Fatalf("unexpected: %v %v", v, ok)
	}
}

func TestPipeline_RunSequential(t *testing.T) {
	p := New("seq",
		&TaskFunc{N: "double", F: func(ctx context.Context, c *PipelineContext, in any) (any, error) {
			n, _ := in.(int)
			return n * 2, nil
		}},
		&TaskFunc{N: "addone", F: func(ctx context.Context, c *PipelineContext, in any) (any, error) {
			n, _ := in.(int)
			return n + 1, nil
		}},
	)
	pctx := NewPipelineContext("seq")
	out, results, err := p.Run(context.Background(), pctx, 5)
	if err != nil {
		t.Fatal(err)
	}
	if out.(int) != 11 {
		t.Fatalf("expected 11, got %v", out)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestPipeline_FailFastAborts(t *testing.T) {
	boom := errors.New("boom")
	p := New("ff",
		&TaskFunc{N: "ok", F: func(ctx context.Context, c *PipelineContext, in any) (any, error) { return "a", nil }},
		&TaskFunc{N: "fail", F: func(ctx context.Context, c *PipelineContext, in any) (any, error) { return nil, boom }},
		&TaskFunc{N: "after", F: func(ctx context.Context, c *PipelineContext, in any) (any, error) {
			t.Fatal("after should not run with FailFast=true")
			return nil, nil
		}},
	)
	p.FailFast = true
	_, _, err := p.Run(context.Background(), NewPipelineContext("ff"), nil)
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom, got %v", err)
	}
}

func TestPipeline_FailOpenContinues(t *testing.T) {
	boom := errors.New("boom")
	called := 0
	p := New("fo",
		&TaskFunc{N: "ok", F: func(ctx context.Context, c *PipelineContext, in any) (any, error) { return "a", nil }},
		&TaskFunc{N: "fail", F: func(ctx context.Context, c *PipelineContext, in any) (any, error) { return nil, boom }},
		&TaskFunc{N: "after", F: func(ctx context.Context, c *PipelineContext, in any) (any, error) {
			called++
			return "b", nil
		}},
	)
	out, _, err := p.Run(context.Background(), NewPipelineContext("fo"), nil)
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom surfaced, got %v", err)
	}
	if called != 1 {
		t.Fatalf("after should still run in fail-open mode, called=%d", called)
	}
	if out.(string) != "b" {
		t.Fatalf("expected final output 'b', got %v", out)
	}
}

func TestConcurrent_AllSucceed(t *testing.T) {
	pctx := NewPipelineContext("c")
	tasks := []Task{
		&TaskFunc{N: "t1", F: func(ctx context.Context, c *PipelineContext, in any) (any, error) {
			time.Sleep(5 * time.Millisecond)
			return "v1", nil
		}},
		&TaskFunc{N: "t2", F: func(ctx context.Context, c *PipelineContext, in any) (any, error) {
			time.Sleep(5 * time.Millisecond)
			return "v2", nil
		}},
	}
	if err := Concurrent(context.Background(), pctx, 2, tasks...); err != nil {
		t.Fatal(err)
	}
	if v, _ := pctx.Get("t1"); v.(string) != "v1" {
		t.Fatalf("t1 output missing")
	}
	if v, _ := pctx.Get("t2"); v.(string) != "v2" {
		t.Fatalf("t2 output missing")
	}
}

func TestConcurrent_AggregatesErrors(t *testing.T) {
	pctx := NewPipelineContext("c")
	tasks := []Task{
		&TaskFunc{N: "t1", F: func(ctx context.Context, c *PipelineContext, in any) (any, error) { return nil, errors.New("e1") }},
		&TaskFunc{N: "t2", F: func(ctx context.Context, c *PipelineContext, in any) (any, error) { return nil, errors.New("e2") }},
	}
	err := Concurrent(context.Background(), pctx, 4, tasks...)
	if err == nil {
		t.Fatal("expected aggregated error")
	}
}

func TestPipeline_StageResultTiming(t *testing.T) {
	p := New("timing",
		&TaskFunc{N: "slow", F: func(ctx context.Context, c *PipelineContext, in any) (any, error) {
			time.Sleep(5 * time.Millisecond)
			return nil, nil
		}},
	)
	_, results, err := p.Run(context.Background(), NewPipelineContext("timing"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Ended.Before(results[0].Started) {
		t.Fatal("end before start")
	}
}
