use std::error;
use tui::{
    layout::{Alignment},
    widgets::{Block, BorderType, Borders},
    Terminal,
};


use core::result::Result;
use std::io::Error;
use tui::backend::Backend;
use tui::buffer::Cell;
use tui::layout::Rect;

extern "C" {
    fn cursorHide();
    fn cursorShow();
    fn cursorSet(x: u16, y: u16);
    fn clearScreen();
    fn getSize() -> (i32, i32, i32, i32);
    fn flush();
    fn draw(x: u16, y: u16);
}

struct HostBackend {}

impl Backend for HostBackend {
    fn draw<'a, I>(&mut self, content: I) -> Result<(), Error>
    where
        I: Iterator<Item = (u16, u16, &'a Cell)>,
    {
	for i in content {
	    let x: u16 = i.0;
	    let y: u16 = i.1;
	    unsafe { draw(x, y) }
	}
        Ok(())
    }

    fn hide_cursor(&mut self) -> Result<(), Error> {
	unsafe { cursorHide() }
        Ok(())
    }

    fn show_cursor(&mut self) -> Result<(), Error> {
	unsafe { cursorShow() }
        Ok(())
    }

    fn get_cursor(&mut self) -> Result<(u16, u16), Error> {
        Ok((100, 100))
    }

    fn set_cursor(&mut self, x: u16, y: u16) -> Result<(), Error> {
	unsafe { cursorSet(x, y) }
        Ok(())
    }

    fn clear(&mut self) -> Result<(), Error> {
	unsafe { clearScreen() }
        Ok(())
    }

    fn size(&self) -> Result<Rect, Error> {
	let ws: (i32, i32, i32, i32);
	unsafe { ws = getSize(); }
        Ok(Rect {
            x: ws.0 as u16,
            y: ws.1 as u16,
            width: ws.2 as u16,
            height: ws.3 as u16,
        })
        // Ok(Rect {
        //     x: 0,
        //     y: 0,
        //     width: 50,
        //     height: 50,
        // })
    }

    fn flush(&mut self) -> Result<(), Error> {
	unsafe { flush() }
        Ok(())
    }
}

fn main() -> Result<(), Box<dyn error::Error>> {
    let backend = HostBackend{};
    let mut terminal = Terminal::new(backend)?;
    let mut i: i32 = 0;
    loop {
	terminal.draw(|f| {
	    let size = f.size();
	    let block = Block::default()
		.borders(Borders::ALL)
		.title("Main block with round corners")
		.title_alignment(Alignment::Center)
		.border_type(BorderType::Rounded);
	    f.render_widget(block, size);
	})?;
	i = i + 1;
	if i > 1000 {
	    break;
	}
    }
    terminal.show_cursor()?;

    Ok(())
}

struct Unused<T>(T);

#[no_mangle]
pub extern "C" fn run() {
    // unsafe { cursorSet(10, 10) }
    Unused(main());
}
