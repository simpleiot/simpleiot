module Components.NodeDb exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style exposing (colors)


view : NodeOptions msg -> Element msg
view o =
    let
        labelWidth =
            150

        opts =
            oToInputO o labelWidth

        textInput =
            NodeInputs.nodeTextInput opts "" 0
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.database
            , text <|
                Point.getText o.node.points "" 0 Point.typeDescription
            ]
            :: (if o.expDetail then
                    [ text "InfluxDb 2.0 Database"
                    , textInput Point.typeDescription "Description" ""
                    , textInput Point.typeURI "URL" "https://myserver:8086"
                    , textInput Point.typeOrg "Organization" "org name"
                    , textInput Point.typeBucket "Bucket" "bucket name"
                    , textInput Point.typeAuthToken "Auth Token" ""
                    ]

                else
                    []
               )
