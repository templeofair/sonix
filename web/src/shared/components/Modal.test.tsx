import { useState } from 'react'
import { describe, it, expect, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import Modal from './Modal'

function Fixture({ onClose, dismiss }: { onClose: () => void; dismiss?: 'click' | 'mousedown' }) {
  return (
    <Modal onClose={onClose} labelledBy="fixture-title" dismiss={dismiss}>
      <h2 id="fixture-title">Fixture</h2>
      <button type="button">First</button>
      <button type="button">Last</button>
    </Modal>
  )
}

describe('Modal', () => {
  it('closes on Escape', async () => {
    const onClose = vi.fn()
    render(<Fixture onClose={onClose} />)
    await userEvent.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('focuses the panel and keeps Tab inside it', () => {
    render(<Fixture onClose={vi.fn()} />)
    expect(screen.getByRole('dialog')).toHaveFocus()

    screen.getByRole('button', { name: 'Last' }).focus()
    fireEvent.keyDown(window, { key: 'Tab' })
    expect(screen.getByRole('button', { name: 'First' })).toHaveFocus()

    fireEvent.keyDown(window, { key: 'Tab', shiftKey: true })
    expect(screen.getByRole('button', { name: 'Last' })).toHaveFocus()
  })

  it('returns focus to the opener on unmount', async () => {
    function Host() {
      const [open, setOpen] = useState(false)
      return (
        <>
          <button type="button" onClick={() => setOpen(true)}>
            Open
          </button>
          {open ? (
            <Modal onClose={() => setOpen(false)} labelledBy="host-title">
              <h2 id="host-title">Host</h2>
              <button type="button" onClick={() => setOpen(false)}>
                Close
              </button>
            </Modal>
          ) : null}
        </>
      )
    }
    render(<Host />)
    const opener = screen.getByRole('button', { name: 'Open' })
    await userEvent.click(opener)
    await userEvent.click(screen.getByRole('button', { name: 'Close' }))
    expect(opener).toHaveFocus()
  })

  it('dismisses from the backdrop, and never from the panel itself', async () => {
    const onClose = vi.fn()
    const { unmount } = render(<Fixture onClose={onClose} />)
    const backdrop = screen.getByRole('dialog').parentElement as HTMLElement
    await userEvent.click(backdrop)
    expect(onClose).toHaveBeenCalledTimes(1)
    await userEvent.click(screen.getByRole('dialog'))
    expect(onClose).toHaveBeenCalledTimes(1)
    unmount()

    const onCloseMouse = vi.fn()
    render(<Fixture onClose={onCloseMouse} dismiss="mousedown" />)
    fireEvent.mouseDown(screen.getByRole('dialog'))
    expect(onCloseMouse).not.toHaveBeenCalled()
    fireEvent.mouseDown(screen.getByRole('dialog').parentElement as HTMLElement)
    expect(onCloseMouse).toHaveBeenCalledTimes(1)
  })

  it('dismiss=none keeps the dialog on backdrop click; overlay mousedown still notifies', async () => {
    const onClose = vi.fn()
    const onOverlayMouseDown = vi.fn()
    function Host() {
      return (
        <Modal
          onClose={onClose}
          labelledBy="none-title"
          dismiss="none"
          onOverlayMouseDown={onOverlayMouseDown}
        >
          <h2 id="none-title">None</h2>
          <button type="button">Ok</button>
        </Modal>
      )
    }
    render(<Host />)
    const backdrop = screen.getByRole('dialog').parentElement as HTMLElement
    await userEvent.click(backdrop)
    expect(onClose).not.toHaveBeenCalled()
    fireEvent.mouseDown(backdrop)
    expect(onOverlayMouseDown).toHaveBeenCalled()
  })

  it('portals onto document.body so nested stacking contexts cannot cover it', () => {
    function Nested() {
      return (
        <div className="relative z-0" data-testid="stack-trap">
          <Modal onClose={vi.fn()} labelledBy="portal-title">
            <h2 id="portal-title">Portaled</h2>
          </Modal>
        </div>
      )
    }
    render(<Nested />)
    const dialog = screen.getByRole('dialog')
    expect(dialog.parentElement?.parentElement).toBe(document.body)
    expect(screen.getByTestId('stack-trap').contains(dialog)).toBe(false)
  })
})
