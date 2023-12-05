module Components.NodeRaw exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import UI.Style as Style


view : NodeOptions msg -> Element msg
view o =
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color Style.colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10, Background.color Style.colors.yellow ]
            [ Element.text <| "Node type: " ++ o.node.typ
            ]
            :: (if o.expDetail then
                    [ viewPoints o.node.points
                    ]

                else
                    []
               )


viewPoints : List Point.Point -> Element msg
viewPoints pts =
    table [ padding 7 ]
        { data = List.map Point.renderPoint2 pts
        , columns =
            let
                cell =
                    el [ paddingXY 15 5, Border.width 1 ]
            in
            [ { header = cell <| el [ Font.bold, centerX ] <| text "Point"
              , width = fill
              , view = \m -> cell <| text m.desc
              }
            , { header = cell <| el [ Font.bold, centerX ] <| text "Value"
              , width = fill
              , view = \m -> cell <| el [ alignRight ] <| text m.value
              }
            ]
        }
