module UI.Form exposing (button, buttonRow, label, viewTextProperty)

import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Element.Input as Input
import UI.Style as Style


type alias TextProperty msg =
    { name : String
    , value : String
    , action : String -> msg
    }


viewTextProperty : TextProperty msg -> Element msg
viewTextProperty { name, value, action } =
    Input.text
        [ padding 16
        , width (fill |> minimum 150)
        , Border.width 0
        , Border.rounded 0
        , Background.color Style.colors.pale
        , spacing 0
        ]
        { onChange = action
        , text = value
        , placeholder = Nothing
        , label = label Input.labelAbove name
        }


label : (List (Attribute msg) -> Element msg -> Input.Label msg) -> (String -> Input.Label msg)
label kind =
    kind
        [ padding 16
        , Font.italic
        , Font.color Style.colors.gray
        ]
        << text


buttonRow : List (Element msg) -> Element msg
buttonRow =
    row
        [ Font.size 16
        , Font.bold
        , width fill
        , padding 16
        , spacing 16
        ]


button : String -> Color -> msg -> Element msg
button lbl color action =
    Input.button
        (Style.button color)
        { onPress = Just action
        , label = el [ centerX ] <| text lbl
        }
