import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryManagementPage } from '@/app/(dashboard)/memories/page';
import { MemoryForm } from '@/components/memories/memory-form';
import { MemoryTable } from '@/components/memories/memory-table';

// Mock API functions
const mockMemoriesApi = {
  list: jest.fn(),
  create: jest.fn(),
  update: jest.fn(),
  delete: jest.fn(),
  batchDelete: jest.fn(),
  batchUpdate: jest.fn(),
};

const mockToast = jest.fn();

jest.mock('@/lib/api', () => ({
  memoriesApi: mockMemoriesApi,
}));

jest.mock('sonner', () => ({
  toast: {
    success: mockToast,
    error: mockToast,
  },
}));

const createQueryClient = () => new QueryClient({
  defaultOptions: {
    queries: { retry: false },
    mutations: { retry: false },
  },
});

describe('Memory Management Page', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = createQueryClient();
    jest.clearAllMocks();
  });

  it('renders memory list with loading state', () => {
    mockMemoriesApi.list.mockReturnValue(Promise.resolve({ data: [] }));
    
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryManagementPage />
      </QueryClientProvider>
    );

    expect(screen.getByText('Memories')).toBeInTheDocument();
    expect(screen.getByText('Loading...')).toBeInTheDocument();
  });

  it('displays memories data', async () => {
    const mockMemories = [
      {
        id: '1',
        content: 'Test memory 1',
        type: 'user',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      },
      {
        id: '2',
        content: 'Test memory 2',
        type: 'user',
        created_at: '2024-01-02T00:00:00Z',
        updated_at: '2024-01-02T00:00:00Z',
      },
    ];

    mockMemoriesApi.list.mockReturnValue(Promise.resolve({ data: mockMemories }));
    
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryManagementPage />
      </QueryClientProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Test memory 1')).toBeInTheDocument();
      expect(screen.getByText('Test memory 2')).toBeInTheDocument();
    });
  });

  it('opens create dialog when clicking add button', async () => {
    mockMemoriesApi.list.mockReturnValue(Promise.resolve({ data: [] }));
    
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryManagementPage />
      </QueryClientProvider>
    );

    const addButton = screen.getByText('Add Memory');
    fireEvent.click(addButton);

    await waitFor(() => {
      expect(screen.getByText('Create Memory')).toBeInTheDocument();
    });
  });
});

describe('Memory Form', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = createQueryClient();
    jest.clearAllMocks();
  });

  it('renders form with all fields', () => {
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryForm />
      </QueryClientProvider>
    );

    expect(screen.getByLabelText('Content')).toBeInTheDocument();
    expect(screen.getByLabelText('Type')).toBeInTheDocument();
    expect(screen.getByLabelText('Category')).toBeInTheDocument();
    expect(screen.getByLabelText('Importance')).toBeInTheDocument();
    expect(screen.getByText('Create Memory')).toBeInTheDocument();
  });

  it('submits form with valid data', async () => {
    const mockCreate = jest.fn().mockReturnValue(Promise.resolve({ success: true }));
    mockMemoriesApi.create = mockCreate;

    render(
      <QueryClientProvider client={queryClient}>
        <MemoryForm />
      </QueryClientProvider>
    );

    fireEvent.change(screen.getByLabelText('Content'), {
      target: { value: 'Test memory content' },
    });
    
    fireEvent.change(screen.getByLabelText('Type'), {
      target: { value: 'user' },
    });

    fireEvent.click(screen.getByText('Create Memory'));

    await waitFor(() => {
      expect(mockCreate).toHaveBeenCalledWith({
        content: 'Test memory content',
        type: 'user',
        metadata: {},
      });
    });

    expect(mockToast).toHaveBeenCalledWith('Memory created successfully');
  });

  it('displays validation errors on invalid form', async () => {
    render(
      <QueryClientProvider client={queryClient}>
        <MemoryForm />
      </QueryClientProvider>
    );

    // Try to submit empty form
    fireEvent.click(screen.getByText('Create Memory'));

    await waitFor(() => {
      expect(screen.getByText('Content is required')).toBeInTheDocument();
    });
  });
});

describe('Memory Table', () => {
  let queryClient: QueryClient;

  beforeEach(() => {
    queryClient = createQueryClient();
    jest.clearAllMocks();
  });

  it('renders table with memories', () => {
    const mockMemories = [
      {
        id: '1',
        content: 'Test memory 1',
        type: 'user',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      },
    ];

    render(
      <QueryClientProvider client={queryClient}>
        <MemoryTable memories={mockMemories} onDelete={jest.fn()} />
      </QueryClientProvider>
    );

    expect(screen.getByText('Test memory 1')).toBeInTheDocument();
    expect(screen.getByText('user')).toBeInTheDocument();
  });

  it('calls delete function when delete button is clicked', async () => {
    const mockDelete = jest.fn();
    const mockMemories = [
      {
        id: '1',
        content: 'Test memory 1',
        type: 'user',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      },
    ];

    render(
      <QueryClientProvider client={queryClient}>
        <MemoryTable memories={mockMemories} onDelete={mockDelete} />
      </QueryClientProvider>
    );

    const deleteButton = screen.getByText('Delete');
    fireEvent.click(deleteButton);

    // Should show confirmation dialog
    expect(screen.getByText('Are you sure you want to delete this memory?')).toBeInTheDocument();
    
    const confirmButton = screen.getByText('Delete');
    fireEvent.click(confirmButton);

    await waitFor(() => {
      expect(mockDelete).toHaveBeenCalledWith('1');
    });
  });
});