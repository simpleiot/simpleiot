module Components.NodeUser exposing (view)

import Api.Node exposing (Node)
import Api.Point as Point exposing (Point)
import Element exposing (..)
import Element.Border as Border
import Time
import UI.Form as Form
import UI.Icon as Icon
import UI.Style exposing (colors)


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
