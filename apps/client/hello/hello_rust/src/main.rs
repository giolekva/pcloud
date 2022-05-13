use wasm_bindgen::prelude::*;
use crossterm::{
    event::{DisableMouseCapture, EnableMouseCapture },
    execute,
    terminal::{disable_raw_mode, enable_raw_mode, EnterAlternateScreen, LeaveAlternateScreen},
};
use std::{error::Error, io};
use tui::{
    backend::{CrosstermBackend},
    layout::{Alignment},
    widgets::{Block, BorderType, Borders},
    Terminal,
};

fn main() -> Result<(), Box<dyn Error>> {
    enable_raw_mode()?;
    let mut stdout = io::stdout();
    execute!(stdout, EnterAlternateScreen, EnableMouseCapture)?;

    let backend = CrosstermBackend::new(stdout);
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
    // restore terminal
    disable_raw_mode()?;
    execute!(
        terminal.backend_mut(),
        LeaveAlternateScreen,
        DisableMouseCapture
    )?;
    terminal.show_cursor()?;

    Ok(())
}

// foo

#[wasm_bindgen]
extern "C" {
    fn hello(s: &str);
}

struct Unused<T>(T);

#[no_mangle]
pub extern "C" fn run() {
    Unused(main());
}
