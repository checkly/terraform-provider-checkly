resource "checkly_status_page_v3" "example" {
  name          = "Example Application"
  url           = "my-example-status-page"
  default_theme = "DARK"
  support_link  = "mailto:support@example.com"

  # Optional custom colors (requires custom theme colors on your plan). The
  # complete palette must be given; leave the block out for the defaults.
  theme_colors {
    light {
      body_background_color           = "#F9FAFB"
      header_background_color         = "#FFFFFF"
      header_font_color               = "#151A1E"
      title_font_color                = "#212930"
      body_font_color                 = "#475766"
      body_font_color_muted           = "#60758A"
      navigation_font_color           = "#151A1E"
      link_font_color                 = "#FF0000"
      card_background_color           = "#FFFFFF"
      border_color                    = "#E0E5EB"
      primary_button_background_color = "#151A1E"
      primary_button_font_color       = "#FFFFFF"
    }
    dark {
      body_background_color           = "#14171C"
      header_background_color         = "#171B21"
      header_font_color               = "#FFFFFF"
      title_font_color                = "#ECEEF2"
      body_font_color                 = "#C6CDD7"
      body_font_color_muted           = "#A3B3C2"
      navigation_font_color           = "#FFFFFF"
      link_font_color                 = "#248AFF"
      card_background_color           = "#171B21"
      border_color                    = "#242B34"
      primary_button_background_color = "#242B34"
      primary_button_font_color       = "#FFFFFF"
    }
  }
}

# The page structure is declared with components: GROUPs contain SERVICEs.
resource "checkly_status_page_v3_component" "services" {
  status_page_id = checkly_status_page_v3.example.id
  type           = "GROUP"
  name           = "Services"
  display_order  = 0
}

resource "checkly_status_page_v3_component" "api" {
  status_page_id = checkly_status_page_v3.example.id
  name           = "API"
  display_order  = 1
  parent_id      = checkly_status_page_v3_component.services.id
}

resource "checkly_status_page_v3_component" "database" {
  status_page_id = checkly_status_page_v3.example.id
  name           = "Database"
  display_order  = 2
  parent_id      = checkly_status_page_v3_component.services.id
}
