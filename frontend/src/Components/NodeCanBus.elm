module Components.NodeCanBus exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view : NodeOptions msg -> Element msg
view o =
    let
        labelWidth =
            180

        opts =
            oToInputO o labelWidth

        textInput =
            NodeInputs.nodeTextInput opts ""

        
        checkboxInput =
            NodeInputs.nodeCheckboxInput opts ""

        disabled =
            Point.getBool o.node.points Point.typeDisable ""

    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.serialDev
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            , viewIf disabled <| text "(disabled)"
            ]
            :: (if o.expDetail then
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typeInterface "Interface" "can0"
                    , textInput Point.typeBaud "Bus speed" "250K"
                    , checkboxInput Point.typeDisable "Disable"
                    , viewPoints <| Point.filterSpecialPoints <| List.sortWith Point.sort o.node.points
                    ]

                else
                    []
               )


viewPoints : List Point.Point -> Element msg
viewPoints ios =
    column
        [ padding 16
        , spacing 6
        ]
    <|
        List.map (Point.renderPoint >> text) ios
