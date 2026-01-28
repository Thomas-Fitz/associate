import { describe, it, expect } from 'vitest'
import { TerminalQueries } from './cypher-builder'

describe('TerminalQueries', () => {
  describe('listByZone', () => {
    it('should generate correct query with zone ID', () => {
      const query = TerminalQueries.listByZone('zone-123')
      
      expect(query).toContain('MATCH (term:Terminal)')
      expect(query).toContain('BELONGS_TO')
      expect(query).toContain("id: 'zone-123'")
      expect(query).toContain('RETURN term')
      expect(query).toContain('ORDER BY term.created_at DESC')
    })

    it('should escape special characters in zone ID', () => {
      const query = TerminalQueries.listByZone("zone-with'quote")
      
      expect(query).toContain("\\'quote")
    })
  })

  describe('countInZone', () => {
    it('should generate correct AGE-compatible count query', () => {
      const query = TerminalQueries.countInZone('zone-123')
      
      // Should use MATCH + OPTIONAL MATCH + WITH pattern for AGE compatibility
      // Must include z in WITH clause for AGE to parse correctly
      expect(query).toContain('MATCH (z:Zone')
      expect(query).toContain('OPTIONAL MATCH (term:Terminal)')
      expect(query).toContain('WITH z, count(term) as terminal_count')
      expect(query).toContain('RETURN terminal_count')
    })

    it('should properly reference zone ID', () => {
      const query = TerminalQueries.countInZone('my-zone-id')
      
      expect(query).toContain("id: 'my-zone-id'")
    })

    it('should escape special characters in zone ID', () => {
      const query = TerminalQueries.countInZone("zone'with\"special")
      
      expect(query).toContain("\\'with")
      expect(query).toContain('\\"special')
    })
  })

  describe('create', () => {
    it('should generate correct create query', () => {
      const query = TerminalQueries.create({
        id: 'term-123',
        name: 'Terminal 1',
        config: '{}',
        state: '{"status":"disconnected"}',
        metadata: '{"ui_x":100,"ui_y":200}'
      })
      
      expect(query).toContain('CREATE (term:Terminal')
      expect(query).toContain("id: 'term-123'")
      expect(query).toContain("name: 'Terminal 1'")
      expect(query).toContain("node_type: 'Terminal'")
      expect(query).toContain('RETURN term')
    })

    it('should include timestamps', () => {
      const query = TerminalQueries.create({
        id: 'term-123',
        name: 'Terminal 1',
        config: '{}',
        state: '{}',
        metadata: '{}'
      })
      
      expect(query).toContain('created_at:')
      expect(query).toContain('updated_at:')
    })
  })

  describe('linkToZone', () => {
    it('should generate correct link query', () => {
      const query = TerminalQueries.linkToZone('term-123', 'zone-456')
      
      expect(query).toContain('MATCH (term:Terminal')
      expect(query).toContain("term-123'")
      expect(query).toContain('MATCH')
      expect(query).toContain('(z:Zone')
      expect(query).toContain("zone-456'")
      expect(query).toContain('CREATE (term)-[:BELONGS_TO]->(z)')
      expect(query).toContain('RETURN term')
    })
  })

  describe('update', () => {
    it('should generate correct update query with sets', () => {
      const sets = ["term.name = 'New Name'", "term.updated_at = '2024-01-01'"]
      const query = TerminalQueries.update('term-123', sets)
      
      expect(query).toContain('MATCH (term:Terminal')
      expect(query).toContain("id: 'term-123'")
      expect(query).toContain("SET term.name = 'New Name'")
      expect(query).toContain("term.updated_at = '2024-01-01'")
      expect(query).toContain('RETURN term')
    })
  })

  describe('delete', () => {
    it('should generate correct delete query', () => {
      const query = TerminalQueries.delete('term-123')
      
      expect(query).toContain('MATCH (term:Terminal')
      expect(query).toContain("id: 'term-123'")
      expect(query).toContain('DETACH DELETE term')
    })
  })

  describe('getById', () => {
    it('should generate correct get by ID query', () => {
      const query = TerminalQueries.getById('term-123')
      
      expect(query).toContain('MATCH (term:Terminal')
      expect(query).toContain("id: 'term-123'")
      expect(query).toContain('RETURN term')
    })
  })
})
