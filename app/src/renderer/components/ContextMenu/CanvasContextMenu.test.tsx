import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { CanvasContextMenu } from './CanvasContextMenu'
import { useAppStore } from '../../stores/appStore'

// Mock the useCanvasNodeCreation hook
const mockCreatePlan = vi.fn()
const mockCreateTask = vi.fn()
const mockCreateMemory = vi.fn()
const mockCanCreateNodeType = vi.fn()
const mockGetCannotCreateReason = vi.fn()

vi.mock('../../hooks/useCanvasNodeCreation', () => ({
  useCanvasNodeCreation: () => ({
    createPlan: mockCreatePlan,
    createTask: mockCreateTask,
    createMemory: mockCreateMemory,
    canCreateNodeType: mockCanCreateNodeType,
    getCannotCreateReason: mockGetCannotCreateReason
  })
}))

describe('CanvasContextMenu', () => {
  const defaultProps = {
    x: 100,
    y: 100,
    canvasX: 200,
    canvasY: 200,
    onClose: vi.fn()
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockCanCreateNodeType.mockReturnValue(true)
    mockGetCannotCreateReason.mockReturnValue(null)
    mockCreatePlan.mockResolvedValue({ id: 'plan-1' })
    mockCreateTask.mockResolvedValue({ id: 'task-1' })
    mockCreateMemory.mockResolvedValue({ id: 'memory-1' })
    
    // Reset store state
    useAppStore.setState({
      selectedZone: {
        id: 'zone-1',
        name: 'Test Zone',
        description: '',
        metadata: {},
        tags: [],
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-01T00:00:00Z',
        planCount: 0,
        taskCount: 0,
        memoryCount: 0,
        plans: [],
        memories: []
      },
      selectedPlan: {
        id: 'plan-1',
        name: 'Test Plan',
        description: '',
        status: 'active',
        metadata: {},
        tags: [],
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-01T00:00:00Z',
        taskCount: 0,
        tasks: []
      }
    })
  })

  it('should render all node type options', () => {
    render(<CanvasContextMenu {...defaultProps} />)
    
    expect(screen.getByText('Add New Plan')).toBeInTheDocument()
    expect(screen.getByText('Add New Task')).toBeInTheDocument()
    expect(screen.getByText('Add Memory')).toBeInTheDocument()
  })

  it('should call createPlan when Add New Plan is clicked', async () => {
    render(<CanvasContextMenu {...defaultProps} />)
    
    const planButton = screen.getByText('Add New Plan').closest('button')!
    fireEvent.click(planButton)
    
    await waitFor(() => {
      expect(mockCreatePlan).toHaveBeenCalledWith({
        position: expect.objectContaining({
          x: expect.any(Number),
          y: expect.any(Number)
        })
      })
    })
    
    expect(defaultProps.onClose).toHaveBeenCalled()
  })

  it('should call createTask when Add New Task is clicked', async () => {
    render(<CanvasContextMenu {...defaultProps} />)
    
    const taskButton = screen.getByText('Add New Task').closest('button')!
    fireEvent.click(taskButton)
    
    await waitFor(() => {
      expect(mockCreateTask).toHaveBeenCalledWith({
        position: expect.objectContaining({
          x: expect.any(Number),
          y: expect.any(Number)
        })
      })
    })
    
    expect(defaultProps.onClose).toHaveBeenCalled()
  })

  it('should show memory submenu with options', () => {
    render(<CanvasContextMenu {...defaultProps} />)
    
    // Hover over Add Memory to open submenu
    const memorySubmenu = screen.getByText('Add Memory').closest('div')!
    fireEvent.mouseEnter(memorySubmenu)
    
    // Check all memory type options are present
    expect(screen.getByText('Note')).toBeInTheDocument()
    expect(screen.getByText('Repository')).toBeInTheDocument()
    expect(screen.getByText('Memory')).toBeInTheDocument()
  })

  it('should call createMemory with Note type when Note is clicked', async () => {
    render(<CanvasContextMenu {...defaultProps} />)
    
    // Hover over Add Memory to open submenu
    const memorySubmenu = screen.getByText('Add Memory').closest('div')!
    fireEvent.mouseEnter(memorySubmenu)
    
    const noteButton = screen.getByText('Note').closest('button')!
    fireEvent.click(noteButton)
    
    await waitFor(() => {
      expect(mockCreateMemory).toHaveBeenCalledWith(
        { position: expect.objectContaining({ x: expect.any(Number), y: expect.any(Number) }) },
        'Note'
      )
    })
    
    expect(defaultProps.onClose).toHaveBeenCalled()
  })

  it('should call createMemory with Repository type when Repository is clicked', async () => {
    render(<CanvasContextMenu {...defaultProps} />)
    
    // Hover over Add Memory to open submenu
    const memorySubmenu = screen.getByText('Add Memory').closest('div')!
    fireEvent.mouseEnter(memorySubmenu)
    
    const repoButton = screen.getByText('Repository').closest('button')!
    fireEvent.click(repoButton)
    
    await waitFor(() => {
      expect(mockCreateMemory).toHaveBeenCalledWith(
        { position: expect.objectContaining({ x: expect.any(Number), y: expect.any(Number) }) },
        'Repository'
      )
    })
  })

  it('should call createMemory with Memory type when Memory is clicked', async () => {
    render(<CanvasContextMenu {...defaultProps} />)
    
    // Hover over Add Memory to open submenu
    const memorySubmenu = screen.getByText('Add Memory').closest('div')!
    fireEvent.mouseEnter(memorySubmenu)
    
    // Need to be more specific since "Memory" appears in both the submenu label and option
    const memoryButtons = screen.getAllByText('Memory')
    const memoryOptionButton = memoryButtons.find(el => el.closest('button'))?.closest('button')!
    fireEvent.click(memoryOptionButton)
    
    await waitFor(() => {
      expect(mockCreateMemory).toHaveBeenCalledWith(
        { position: expect.objectContaining({ x: expect.any(Number), y: expect.any(Number) }) },
        'Memory'
      )
    })
  })

  it('should disable plan option when canCreateNodeType returns false for plan', () => {
    mockCanCreateNodeType.mockImplementation((type: string) => type !== 'plan')
    mockGetCannotCreateReason.mockImplementation((type: string) => 
      type === 'plan' ? 'No zone selected' : null
    )
    
    render(<CanvasContextMenu {...defaultProps} />)
    
    const planButton = screen.getByText('Add New Plan').closest('button')!
    expect(planButton).toBeDisabled()
    expect(screen.getByText('(No zone selected)')).toBeInTheDocument()
  })

  it('should disable task option when canCreateNodeType returns false for task', () => {
    mockCanCreateNodeType.mockImplementation((type: string) => type !== 'task')
    mockGetCannotCreateReason.mockImplementation((type: string) => 
      type === 'task' ? 'No plan selected' : null
    )
    
    render(<CanvasContextMenu {...defaultProps} />)
    
    const taskButton = screen.getByText('Add New Task').closest('button')!
    expect(taskButton).toBeDisabled()
    expect(screen.getByText('(No plan selected)')).toBeInTheDocument()
  })

  it('should use canvas coordinates for positioning when provided', async () => {
    render(<CanvasContextMenu {...defaultProps} canvasX={300} canvasY={400} />)
    
    const planButton = screen.getByText('Add New Plan').closest('button')!
    fireEvent.click(planButton)
    
    await waitFor(() => {
      expect(mockCreatePlan).toHaveBeenCalledWith({
        position: {
          // Canvas coords (300, 400) - half of plan size (400/2=200, 300/2=150) = (100, 250)
          x: 100,
          y: 250
        }
      })
    })
  })

  it('should fall back to screen coordinates when canvas coordinates not provided', async () => {
    render(<CanvasContextMenu {...defaultProps} canvasX={undefined} canvasY={undefined} />)
    
    const taskButton = screen.getByText('Add New Task').closest('button')!
    fireEvent.click(taskButton)
    
    await waitFor(() => {
      expect(mockCreateTask).toHaveBeenCalledWith({
        position: {
          // Screen coords (100, 100) - half of task size (250/2=125, 150/2=75) = (-25, 25)
          x: -25,
          y: 25
        }
      })
    })
  })

  it('should not include Zone option in the menu', () => {
    render(<CanvasContextMenu {...defaultProps} />)
    
    expect(screen.queryByText('Add New Zone')).not.toBeInTheDocument()
    expect(screen.queryByText('Add Zone')).not.toBeInTheDocument()
  })
})
