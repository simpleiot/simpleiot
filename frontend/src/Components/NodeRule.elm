module Components.NodeRule exposing (view)

import Api.Node exposing (Node)
import Api.Point as Point exposing (Point)
import Element exposing (..)
import Element.Background as Background
import Element.Border as Border
import Element.Font as Font
import Time
import UI.Form as Form
import UI.Icon as Icon
import UI.Style as Style exposing (colors)


view :
    { isRoot : Bool
    , now : Time.Posix
    , zone : Time.Zone
    , modified : Bool
    , expDetail : Bool
    , parent : Maybe Node
    , node : Node
    , onEditNodePoint : Point -> msg
    }
    -> Element msg
view o =
    let
        textInput =
            Form.nodeTextInput
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , labelWidth = 100
                }
                ""
                0

        active =
            Point.getBool o.node.points "" 0 Point.typeActive

        descBackgroundColor =
            if active then
                Style.colors.blue

            else
                Style.colors.none

        descTextColor =
            if active then
                Style.colors.white

            else
                Style.colors.black
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.list
            , el [ Background.color descBackgroundColor, Font.color descTextColor ] <|
                text <|
                    Point.getText o.node.points "" 0 Point.typeDescription
            ]
            :: (if o.expDetail then
                    [ textInput Point.typeDescription "Description"
                    ]

                else
                    []
               )
