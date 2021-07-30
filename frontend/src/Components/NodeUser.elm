module Components.NodeUser exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions)
import Element exposing (..)
import Element.Border as Border
import UI.Form as Form
import UI.Icon as Icon
import UI.Style exposing (colors)


view : NodeOptions msg -> Element msg
view o =
    let
        labelWidth =
            100

        textInput =
            Form.nodeTextInput
                { onEditNodePoint = o.onEditNodePoint
                , node = o.node
                , now = o.now
                , labelWidth = labelWidth
                }
                ""
                0

        textInputLowerCase =
            Form.nodeTextInput
                { onEditNodePoint =
                    \p ->
                        o.onEditNodePoint { p | text = String.toLower p.text }
                , node = o.node
                , now = o.now
                , labelWidth = labelWidth
                }
                ""
                0
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.user
            , text <|
                Point.getText o.node.points "" 0 Point.typeFirstName
                    ++ " "
                    ++ Point.getText o.node.points "" 0 Point.typeLastName
            ]
            :: (if o.expDetail then
                    [ textInput Point.typeFirstName "First Name"
                    , textInput Point.typeLastName "Last Name"
                    , textInputLowerCase Point.typeEmail "Email"
                    , textInput Point.typePhone "Phone"
                    , textInput Point.typePass "Pass"
                    ]

                else
                    []
               )
