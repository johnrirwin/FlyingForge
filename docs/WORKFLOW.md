# AI-Augmented Product Development Workflow

A comprehensive guide to building products with AI agents across the entire development lifecycle.

---

## Overview

This workflow transforms how teams build products by leveraging AI agents at every stage—from vision to deployment. The key principle: **code documents every state and interaction better than humans ever could**, so we use working prototypes as the source of truth.

```
Vision → Prototype → PRD → Tickets → Design → Planning → Implementation → Ship
           ↓           ↓        ↓         ↓          ↓            ↓
          v0      ChatPRD   Linear    Figma    Claude Code    GitHub
                  + Notion    MCP      MCP        + MCPs       Actions
```

---

## Stage 1: Vision → Prototype

**Owner:** Executive / Product  
**Goal:** Validate the value proposition with a working prototype

### Tools
| Tool | Purpose | Link |
|------|---------|------|
| **v0.dev** | AI UI generation from natural language | vercel.com/v0 |
| **Bolt.new** | Full-stack app generation | bolt.new |
| **Lovable.dev** | Rapid prototyping | lovable.dev |
| **Replit Agent** | Interactive app building | replit.com |

### Process
1. Write a clear vision statement (1-2 paragraphs)
2. Prompt v0/Bolt iteratively until the prototype feels right
3. Export code to GitHub repo early
4. Conduct customer discovery using the prototype
5. Iterate based on feedback

### Best Practices
- Save every v0 iteration with version notes in commit messages
- The prototype code becomes your "source of truth" for states/interactions
- Don't over-polish—focus on validating core value prop
- Record user sessions with the prototype for later analysis

---

## Stage 2: Prototype → PRD

**Owner:** Product  
**Goal:** Generate comprehensive PRD from working code

### Tools
| Tool | Purpose |
|------|---------|
| **ChatPRD** | AI PRD generation (chatprd.ai) |
| **Notion** | PRD hosting and collaboration |
| **Notion MCP** | Agent access to documentation |
| **GitHub MCP** | Agent access to prototype code |

### The Key Insight

> "Code documents every state interaction better than a human ever could"

Feed your prototype code to an LLM to extract a complete PRD.

### PRD Generation Prompt

```markdown
Analyze this React/TypeScript codebase and generate a PRD that documents:

## Required Sections

1. **User-Facing States & Interactions**
   - Every screen/view in the application
   - All user actions and their outcomes
   - Navigation flows between screens

2. **Data Models & Relationships**
   - All TypeScript interfaces/types
   - API request/response shapes
   - State management patterns

3. **Edge Cases Handled**
   - Error states and error messages
   - Loading states
   - Empty states
   - Offline behavior (if any)

4. **Authentication & Authorization**
   - Login/signup flows
   - Protected routes
   - Permission levels

5. **User Stories**
   - Format: "As a [user], I want to [action] so that [benefit]"
   - Derive from actual implemented features

6. **Acceptance Criteria**
   - Derived from actual code behavior
   - Testable and specific

## Output Format
- Notion-compatible markdown
- Mermaid diagrams for flows
- Tables for data models
```

### MCP Configuration for PRD Generation

```json
{
  "mcpServers": {
    "notion": {
      "command": "npx",
      "args": ["-y", "@notionhq/notion-mcp-server"],
      "env": {
        "NOTION_API_KEY": "your-notion-api-key"
      }
    },
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "your-github-token"
      }
    }
  }
}
```

---

## Stage 3: PRD → Tickets

**Owner:** Product / Engineering  
**Goal:** Break PRD into implementable work items

### Tools
| Tool | Purpose |
|------|---------|
| **Linear** | Issue tracking (superior API/MCP support) |
| **Linear MCP** | Automated ticket creation |
| **GitHub Projects** | Alternative if already invested |

### Ticket Generation Prompt

```markdown
You have access to:
- The PRD at Notion page [ID] via Notion MCP
- The codebase via GitHub MCP

For each feature in the PRD:

1. **Create Linear Project**
   - Name matching the epic/feature area
   - Link to Notion PRD

2. **Break Into Issues**
   - Maximum 4 hours of work each
   - Clear title: "[Area] Verb + noun"
   - Example: "[API] Add aircraft CRUD endpoints"

3. **Issue Content**
   - Acceptance criteria from PRD
   - Technical notes referencing existing code patterns
   - Links to relevant prototype code files

4. **Metadata**
   - Labels: frontend, backend, infra, design
   - Priority based on PRD feature priority
   - Dependencies between issues

5. **Sequencing**
   - Infrastructure first
   - Backend before frontend
   - Core features before enhancements

Use our monorepo structure for file references:
- /apps/web - Frontend
- /apps/api or /server - Backend
- /packages/shared - Shared code
- /infra - Infrastructure as code
```

### Linear MCP Configuration

```json
{
  "mcpServers": {
    "linear": {
      "command": "npx",
      "args": ["-y", "@linear/mcp-server"],
      "env": {
        "LINEAR_API_KEY": "your-linear-api-key"
      }
    }
  }
}
```

---

## Stage 4: Design Validation

**Owner:** Design  
**Goal:** Create final mocks and validate against PRD

### Tools
| Tool | Purpose |
|------|---------|
| **Figma** | Design tool |
| **Figma MCP** | Agent access to designs |
| **Claude Workspace** | Team AI with MCP access |
| **Anima/Locofy** | Figma to code (optional) |

### Workflow

```
v0 Prototype 
    ↓
Designer creates Figma mocks using design system
    ↓
Agent validates against PRD acceptance criteria
    ↓
Agent flags gaps/inconsistencies
    ↓
Agent updates Linear tickets with design links
    ↓
Design review with stakeholders
```

### Design Validation Prompt

```markdown
You have access to:
- Figma file [ID] via Figma MCP
- PRD in Notion [page ID] via Notion MCP
- Linear project [ID] via Linear MCP

For each screen in Figma:

1. **Map to PRD**
   - Which user story does this screen fulfill?
   - Which acceptance criteria are addressed?

2. **State Coverage**
   - [ ] Empty state
   - [ ] Loading state
   - [ ] Error state
   - [ ] Success/populated state
   - [ ] Edge cases (long text, missing data, etc.)

3. **Design System Compliance**
   - Correct color tokens used?
   - Typography scale followed?
   - Spacing system applied?
   - Component library used correctly?

4. **Gap Analysis**
   - PRD requirements missing from designs
   - Interactions not specified
   - Responsive breakpoints needed

5. **Update Linear**
   - Add Figma frame links to relevant tickets
   - Create new tickets for missing designs
   - Flag blocked tickets
```

### Figma MCP Configuration

```json
{
  "mcpServers": {
    "figma": {
      "command": "npx",
      "args": ["-y", "@anthropic/figma-mcp"],
      "env": {
        "FIGMA_ACCESS_TOKEN": "your-figma-token"
      }
    }
  }
}
```

---

## Stage 5: Technical Planning

**Owner:** Engineering Leads  
**Goal:** Plan implementation with COTS selection and sequencing

### Tools
| Tool | Purpose |
|------|---------|
| **Claude Code** | Planning and implementation |
| **Devin** | Autonomous multi-file changes |
| **Factory** | Enterprise coding agents |
| **Cursor** | AI-native IDE |

### The "Agent Harness" Concept

Agent harnesses are guidance documents that constrain and direct AI agents. They are critical for consistent, high-quality output.

#### Example: AGENT.md

```markdown
# Agent Guidance for [Project Name]

## Tech Stack Decisions (COTS)

| Category | Choice | Notes |
|----------|--------|-------|
| Database | PostgreSQL 16 | Primary data store |
| Cache | Redis 7 | Session, rate limiting |
| Queue | AWS MSK (Kafka) | Event streaming |
| Search | OpenSearch | Full-text search |
| Auth | Custom JWT | See /server/internal/auth |
| Monitoring | Prometheus + Grafana | Metrics and dashboards |

## Architecture Patterns

### Backend
- Clean architecture: handlers → services → stores
- Dependency injection via constructors
- Repository pattern for data access
- Domain events for cross-service communication

### Frontend
- React with functional components
- TanStack Query for server state
- Context API for UI state (no Redux)
- Optimistic updates where appropriate

### API Design
- RESTful with OpenAPI 3.0 spec
- Consistent error response format
- Pagination via cursor, not offset

## Code Standards

### Go
- Follow /server/docs/ARCHITECTURE.md
- Table-driven tests
- Structured logging with slog
- No ORM—use sqlc for type-safe SQL

### TypeScript
- Strict mode enabled
- Functional components only
- Co-locate tests with components
- Use TypeScript interfaces, not classes

## What NOT To Do

- ❌ Don't introduce new ORMs
- ❌ Don't add state management libraries
- ❌ Don't create new services without ADR
- ❌ Don't bypass the service layer from handlers
- ❌ Don't use `any` type in TypeScript
```

### Technical Planning Prompt

```markdown
You have access to:
- Codebase via GitHub MCP
- PRD via Notion MCP
- Design specs via Figma MCP
- Tickets via Linear MCP
- AGENT.md guidance in the repository

For Linear project [ID], create an implementation plan:

1. **Analyze Requirements**
   - Read each ticket's acceptance criteria
   - Cross-reference with PRD and designs
   - Identify implicit requirements

2. **Map to Existing Code**
   - Find similar patterns already implemented
   - Identify reusable components/services
   - Note code that needs refactoring

3. **Infrastructure Needs**
   - New database tables/migrations
   - New API endpoints
   - New background jobs/queues
   - Third-party integrations

4. **Sequence Work**
   - Database migrations first
   - Backend services second
   - API endpoints third
   - Frontend last
   - Account for dependencies

5. **Estimate Complexity**
   - S: < 2 hours, straightforward
   - M: 2-4 hours, some complexity
   - L: 4-8 hours, significant work
   - XL: > 8 hours, should be split

6. **Document on Tickets**
   - Add implementation notes
   - Link related tickets
   - Flag architectural decisions for review

7. **Human Review Triggers**
   - New external dependencies
   - Security-sensitive changes
   - Breaking API changes
   - Performance-critical paths
```

---

## Stage 6: Autonomous Implementation

**Owner:** Engineering (with AI agents)  
**Goal:** Ship code with human oversight at key points

### Tools
| Tool | Purpose |
|------|---------|
| **Claude Code** | Implementation with MCP access |
| **Devin** | Complex multi-file autonomous work |
| **GitHub Actions** | CI/CD pipeline |
| **CodeRabbit** | AI PR review |

### Execution Loop

```
For each ticket in priority order:
    │
    ├─→ Agent reads ticket + PRD + Figma
    │
    ├─→ Agent creates feature branch
    │
    ├─→ Agent implements with tests
    │       │
    │       └─→ Follows AGENT.md patterns
    │
    ├─→ Agent opens PR with context
    │       │
    │       ├─→ Links to Linear ticket
    │       ├─→ Describes changes
    │       └─→ Notes any deviations
    │
    ├─→ CI runs
    │       │
    │       ├─→ Pass → Continue
    │       └─→ Fail → Agent fixes
    │
    ├─→ Human reviews
    │       │
    │       ├─→ Approve → Merge
    │       └─→ Request changes → Agent addresses
    │
    └─→ Merge triggers Linear ticket closure
```

### Implementation Prompt

```markdown
Implement Linear ticket [ID].

## Context Gathering
1. Read the ticket description and acceptance criteria
2. Check linked PRD sections via Notion MCP
3. Review Figma designs via Figma MCP
4. Read AGENT.md for patterns and constraints

## Implementation Steps
1. Create branch: feature/[ticket-id]-short-description
2. Write failing tests for acceptance criteria
3. Implement the feature following existing patterns
4. Ensure tests pass
5. Run linter and fix issues
6. Commit with conventional commit message

## PR Description Template
```
## Summary
[One paragraph describing what this PR does]

## Linear Ticket
[Link to ticket]

## Changes
- [Bullet list of changes]

## Testing
- [How to test manually]
- [Automated test coverage]

## Screenshots (if UI)
[Before/after or new screens]
```

## Quality Checklist
- [ ] Tests cover acceptance criteria
- [ ] No TypeScript errors
- [ ] No linter warnings
- [ ] Follows AGENT.md patterns
- [ ] PR is focused (single concern)
```

---

## Complete MCP Configuration

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": {
        "GITHUB_TOKEN": "ghp_..."
      }
    },
    "linear": {
      "command": "npx",
      "args": ["-y", "@linear/mcp-server"],
      "env": {
        "LINEAR_API_KEY": "lin_api_..."
      }
    },
    "notion": {
      "command": "npx",
      "args": ["-y", "@notionhq/notion-mcp-server"],
      "env": {
        "NOTION_API_KEY": "secret_..."
      }
    },
    "figma": {
      "command": "npx",
      "args": ["-y", "@anthropic/figma-mcp"],
      "env": {
        "FIGMA_ACCESS_TOKEN": "figd_..."
      }
    },
    "postgres": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-postgres"],
      "env": {
        "DATABASE_URL": "postgresql://..."
      }
    },
    "filesystem": {
      "command": "npx",
      "args": [
        "-y",
        "@modelcontextprotocol/server-filesystem",
        "/path/to/monorepo"
      ]
    }
  }
}
```

---

## Critical Success Factors

| Factor | Why It Matters |
|--------|----------------|
| **Monorepo** | Single source of truth; agents see full context |
| **Strong markdown docs** | AGENT.md, ARCHITECTURE.md guide agent behavior |
| **Atomic tickets** | 2-4 hour chunks agents can complete autonomously |
| **Design system** | Constrained choices = better agent output |
| **Type safety** | TypeScript/Go give agents compile-time feedback |
| **Good tests** | Agents can verify their own work |
| **CI/CD pipeline** | Automated quality gates catch issues early |
| **Conventional commits** | Consistent history agents can learn from |

---

## Team Role Evolution

Every team member becomes a "thinker" who guides AI agents:

| Role | Traditional | AI-Augmented |
|------|-------------|--------------|
| **PM** | Write specs manually | Write prompts that extract PRDs from prototypes |
| **Designer** | Create mockups | Create structured Figma files + validation prompts |
| **Tech Lead** | Review all code | Write AGENT.md, review agent PRs, handle edge cases |
| **Engineer** | Write all code | Write implementation guidance, pair with agents |
| **QA** | Manual testing | Write test generation prompts, validate coverage |

### Key Mindset Shifts

1. **From writing to reviewing** - You'll review more code than you write
2. **From doing to directing** - Clear instructions beat implicit knowledge
3. **From silos to systems** - Everyone contributes to the agent harness
4. **From perfection to iteration** - Ship fast, fix with agents

---

## Getting Started Checklist

- [ ] Set up Linear with API access
- [ ] Configure GitHub MCP in your development environment
- [ ] Write your first AGENT.md with tech stack decisions
- [ ] Create a Notion workspace for PRDs
- [ ] Document your design system for agent consumption
- [ ] Set up CI/CD with quality gates
- [ ] Try the prototype → PRD flow with one feature
- [ ] Iterate on prompts and save what works

---

## Resources

- [Model Context Protocol](https://modelcontextprotocol.io/)
- [Linear API Documentation](https://developers.linear.app/)
- [Notion API Documentation](https://developers.notion.com/)
- [Figma API Documentation](https://www.figma.com/developers/api)
- [v0.dev](https://v0.dev/)
- [ChatPRD](https://chatprd.ai/)

---

*Last updated: February 2026*
