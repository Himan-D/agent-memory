import { rest } from 'msw';
import { setupServer } from 'msw/node';
import { renderHook, waitFor } from '@testing-library/react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { memoriesApi, usersApi, skillsApi } from '@/lib/api';

// Mock API response types
interface Memory {
  id: string;
  content: string;
  type: string;
  created_at: string;
  updated_at: string;
}

interface User {
  id: string;
  email: string;
  name: string;
  role: string;
  created_at: string;
}

interface Skill {
  id: string;
  name: string;
  trigger: string;
  action: string;
  created_at: string;
}

// Mock server setup
const handlers = [
  // Memories API endpoints
  rest.get('/api/memories', (req, res, ctx) => {
    return res(
      ctx.status(200),
      ctx.json({
        data: [
          {
            id: '1',
            content: 'Test memory 1',
            type: 'user',
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
          },
        ],
      })
    );
  }),

  rest.post('/api/memories', async (req, res, ctx) => {
    const body = await req.json();
    return res(
      ctx.status(201),
      ctx.json({
        data: {
          id: 'new-memory',
          content: body.content,
          type: body.type,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      })
    );
  }),

  rest.put('/api/memories/:id', async (req, res, ctx) => {
    const { id } = req.params;
    const body = await req.json();
    return res(
      ctx.status(200),
      ctx.json({
        data: {
          id,
          content: body.content,
          type: body.type,
          created_at: '2024-01-01T00:00:00Z',
          updated_at: new Date().toISOString(),
        },
      })
    );
  }),

  rest.delete('/api/memories/:id', (req, res, ctx) => {
    return res(ctx.status(200));
  }),

  // Users API endpoints
  rest.get('/api/users', (req, res, ctx) => {
    return res(
      ctx.status(200),
      ctx.json({
        data: [
          {
            id: '1',
            email: 'user@example.com',
            name: 'Test User',
            role: 'member',
            created_at: '2024-01-01T00:00:00Z',
          },
        ],
      })
    );
  }),

  rest.post('/api/users', async (req, res, ctx) => {
    const body = await req.json();
    return res(
      ctx.status(201),
      ctx.json({
        data: {
          id: 'new-user',
          email: body.email,
          name: body.name,
          role: body.role,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      })
    );
  }),

  // Skills API endpoints
  rest.get('/api/skills', (req, res, ctx) => {
    return res(
      ctx.status(200),
      ctx.json({
        data: [
          {
            id: '1',
            name: 'Test Skill',
            trigger: 'test',
            action: 'action',
            confidence: 0.8,
            created_at: '2024-01-01T00:00:00Z',
          },
        ],
      })
    );
  }),

  rest.post('/api/skills', async (req, res, ctx) => {
    const body = await req.json();
    return res(
      ctx.status(201),
      ctx.json({
        data: {
          id: 'new-skill',
          name: body.name,
          trigger: body.trigger,
          action: body.action,
          confidence: body.confidence,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
        },
      })
    );
  }),
];

const server = setupServer(...handlers);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

describe('Memory API Integration', () => {
  test('fetches memories list', async () => {
    const { result } = renderHook(() => 
      useQuery({
        queryKey: ['memories'],
        queryFn: memoriesApi.list,
      })
    );

    await waitFor(() => {
      expect(result.current.data).toEqual({
        data: [
          {
            id: '1',
            content: 'Test memory 1',
            type: 'user',
            created_at: '2024-01-01T00:00:00Z',
            updated_at: '2024-01-01T00:00:00Z',
          },
        ],
      });
    });
  });

  test('creates new memory', async () => {
    const { result } = renderHook(() => 
      useMutation({
        mutationFn: memoriesApi.create,
      })
    );

    await result.current.mutateAsync({
      content: 'New memory content',
      type: 'user',
    });

    expect(result.current.data).toEqual({
      data: {
        id: 'new-memory',
        content: 'New memory content',
        type: 'user',
        created_at: expect.any(String),
        updated_at: expect.any(String),
      },
    });
  });

  test('updates memory', async () => {
    const { result } = renderHook(() => 
      useMutation({
        mutationFn: ({ id, data }) => memoriesApi.update(id, data),
      })
    );

    await result.current.mutateAsync({
      id: '1',
      data: {
        content: 'Updated memory content',
        type: 'user',
      },
    });

    expect(result.current.data).toEqual({
      data: {
        id: '1',
        content: 'Updated memory content',
        type: 'user',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: expect.any(String),
      },
    });
  });

  test('deletes memory', async () => {
    const { result } = renderHook(() => 
      useMutation({
        mutationFn: memoriesApi.delete,
      })
    );

    await result.current.mutateAsync('1');

    expect(result.current.data).toBeUndefined();
    expect(result.current.isSuccess).toBe(true);
  });
});

describe('User API Integration', () => {
  test('fetches users list', async () => {
    const { result } = renderHook(() => 
      useQuery({
        queryKey: ['users'],
        queryFn: usersApi.list,
      })
    );

    await waitFor(() => {
      expect(result.current.data).toEqual({
        data: [
          {
            id: '1',
            email: 'user@example.com',
            name: 'Test User',
            role: 'member',
            created_at: '2024-01-01T00:00:00Z',
          },
        ],
      });
    });
  });

  test('creates new user', async () => {
    const { result } = renderHook(() => 
      useMutation({
        mutationFn: usersApi.create,
      })
    );

    await result.current.mutateAsync({
      email: 'newuser@example.com',
      name: 'New User',
      role: 'member',
    });

    expect(result.current.data).toEqual({
      data: {
        id: 'new-user',
        email: 'newuser@example.com',
        name: 'New User',
        role: 'member',
        created_at: expect.any(String),
        updated_at: expect.any(String),
      },
    });
  });
});

describe('Skill API Integration', () => {
  test('fetches skills list', async () => {
    const { result } = renderHook(() => 
      useQuery({
        queryKey: ['skills'],
        queryFn: skillsApi.list,
      })
    );

    await waitFor(() => {
      expect(result.current.data).toEqual({
        data: [
          {
            id: '1',
            name: 'Test Skill',
            trigger: 'test',
            action: 'action',
            confidence: 0.8,
            created_at: '2024-01-01T00:00:00Z',
          },
        ],
      });
    });
  });

  test('creates new skill', async () => {
    const { result } = renderHook(() => 
      useMutation({
        mutationFn: skillsApi.create,
      })
    );

    await result.current.mutateAsync({
      name: 'New Skill',
      trigger: 'new-trigger',
      action: 'new-action',
      confidence: 0.9,
    });

    expect(result.current.data).toEqual({
      data: {
        id: 'new-skill',
        name: 'New Skill',
        trigger: 'new-trigger',
        action: 'new-action',
        confidence: 0.9,
        created_at: expect.any(String),
        updated_at: expect.any(String),
      },
    });
  });
});